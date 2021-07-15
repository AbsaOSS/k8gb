/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/AbsaOSS/k8gb/controllers/providers/metrics"

	str "github.com/AbsaOSS/gopkg/strings"
	k8gbv1beta1 "github.com/AbsaOSS/k8gb/api/v1beta1"
	"github.com/AbsaOSS/k8gb/controllers/depresolver"
	"github.com/AbsaOSS/k8gb/controllers/internal/utils"
	"github.com/AbsaOSS/k8gb/controllers/logging"
	"github.com/AbsaOSS/k8gb/controllers/providers/dns"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	externaldns "sigs.k8s.io/external-dns/endpoint"
)

// GslbReconciler reconciles a Gslb object
type GslbReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Config      *depresolver.Config
	DepResolver *depresolver.DependencyResolver
	DNSProvider dns.Provider
}

const (
	gslbFinalizer                        = "k8gb.absa.oss/finalizer"
	geoStrategy                          = "geoip"
	roundRobinStrategy                   = "roundRobin"
	failoverStrategy                     = "failover"
	primaryGeoTagAnnotation              = "k8gb.io/primary-geotag"
	strategyAnnotation                   = "k8gb.io/strategy"
	dnsTTLSecondsAnnotation              = "k8gb.io/dns-ttl-seconds"
	splitBrainThresholdSecondsAnnotation = "k8gb.io/splitbrain-threshold-seconds"
)

var log = logging.Logger()

var m = metrics.Metrics()

// +kubebuilder:rbac:groups=k8gb.absa.oss,resources=gslbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8gb.absa.oss,resources=gslbs/status,verbs=get;update;patch

// Reconcile runs main reconiliation loop
func (r *GslbReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	result := utils.NewReconcileResultHandler(r.Config.ReconcileRequeueSeconds)
	// Fetch the Gslb instance
	gslb := &k8gbv1beta1.Gslb{}
	err := r.Get(ctx, req.NamespacedName, gslb)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return result.Stop()
		}
		return result.RequeueError(fmt.Errorf("error reading the object (%s)", err))
	}

	err = r.DepResolver.ResolveGslbSpec(ctx, gslb, r.Client)
	if err != nil {
		return result.RequeueError(fmt.Errorf("resolving spec (%s)", err))
	}
	log.Debug().
		Str("Strategy", str.ToString(gslb.Spec.Strategy)).
		Msg("Resolved strategy")
	// == Finalizer business ==

	// Check if the Gslb instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isGslbMarkedToBeDeleted := gslb.GetDeletionTimestamp() != nil
	if isGslbMarkedToBeDeleted {
		// For the legacy reasons, delete all finalizers that corresponds with the slice
		// see: https://sdk.operatorframework.io/docs/upgrading-sdk-version/v1.4.0/#change-your-operators-finalizer-names
		for _, f := range []string{gslbFinalizer, "finalizer.k8gb.absa.oss"} {
			if contains(gslb.GetFinalizers(), f) {
				// Run finalization logic for gslbFinalizer. If the
				// finalization logic fails, don't remove the finalizer so
				// that we can retry during the next reconciliation.
				if err := r.finalizeGslb(gslb); err != nil {
					return result.RequeueError(err)
				}

				// Remove gslbFinalizer. Once all finalizers have been
				// removed, the object will be deleted.
				gslb.SetFinalizers(remove(gslb.GetFinalizers(), f))
				err := r.Update(ctx, gslb)
				if err != nil {
					return result.RequeueError(err)
				}
			}
		}
		log.Info().Msg("reconciler exit")
		return result.Stop()
	}

	// Add finalizer for this CR
	if !contains(gslb.GetFinalizers(), gslbFinalizer) {
		if err := r.addFinalizer(gslb); err != nil {
			return result.RequeueError(err)
		}
	}

	// == Ingress ==========
	ingress, err := r.gslbIngress(gslb)
	if err != nil {
		return result.RequeueError(err)
	}

	err = r.saveIngress(gslb, ingress)
	if err != nil {
		return result.RequeueError(err)
	}

	// == external-dns dnsendpoints CRs ==
	dnsEndpoint, err := r.gslbDNSEndpoint(gslb)
	if err != nil {
		return result.RequeueError(err)
	}

	err = r.DNSProvider.SaveDNSEndpoint(gslb, dnsEndpoint)
	if err != nil {
		return result.RequeueError(err)
	}

	// == handle delegated zone in Edge DNS
	err = r.DNSProvider.CreateZoneDelegationForExternalDNS(gslb)
	if err != nil {
		log.Err(err).Msg("Unable to create zone delegation")
		return result.Requeue()
	}

	// == Status =
	err = r.updateGslbStatus(gslb)
	if err != nil {
		return result.RequeueError(err)
	}

	// == Finish ==========
	// Everything went fine, requeue after some time to catch up
	// with external Gslb status
	// TODO: potentially enhance with smarter reaction to external Event
	m.ReconciliationIncrement()
	return result.Requeue()
}

// SetupWithManager configures controller manager
func (r *GslbReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Figure out Gslb resource name to Reconcile when non controlled Name is updated

	endpointMapHandler := handler.EnqueueRequestsFromMapFunc(
		func(a client.Object) []reconcile.Request {
			gslbList := &k8gbv1beta1.GslbList{}
			opts := []client.ListOption{
				client.InNamespace(a.GetNamespace()),
			}
			c := mgr.GetClient()
			err := c.List(context.TODO(), gslbList, opts...)
			if err != nil {
				log.Info().Msg("Can't fetch gslb objects")
				return nil
			}
			gslbName := ""
			for _, gslb := range gslbList.Items {
				for _, rule := range gslb.Spec.Ingress.Rules {
					for _, path := range rule.HTTP.Paths {
						if path.Backend.ServiceName == a.GetName() {
							gslbName = gslb.Name
						}
					}
				}
			}
			if len(gslbName) > 0 {
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{
						Name:      gslbName,
						Namespace: a.GetNamespace(),
					}},
				}
			}
			return nil
		})

	createGslbFromIngress := func(annotationKey string, annotationValue string, a client.Object, strategy string) {
		log.Info().Msgf("Detected strategy annotation(%s:%s) on Ingress(%s)",
			annotationKey, annotationValue, a.GetName())
		c := mgr.GetClient()
		ingressToReuse := &v1beta1.Ingress{}
		err := c.Get(context.Background(), client.ObjectKey{
			Namespace: a.GetNamespace(),
			Name:      a.GetName(),
		}, ingressToReuse)
		if err != nil {
			log.Info().Msgf("Ingress(%s) does not exist anymore. Skipping Glsb creation...", a.GetName())
			return
		}
		gslbExist := &k8gbv1beta1.Gslb{}
		err = c.Get(context.Background(), client.ObjectKey{
			Namespace: a.GetNamespace(),
			Name:      a.GetName(),
		}, gslbExist)
		if err == nil {
			log.Info().Msgf("Gslb(%s) already exists. Skipping Gslb creation...", gslbExist.Name)
			return
		}
		gslb := &k8gbv1beta1.Gslb{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   a.GetNamespace(),
				Name:        a.GetName(),
				Annotations: a.GetAnnotations(),
			},
			Spec: k8gbv1beta1.GslbSpec{
				Ingress: k8gbv1beta1.FromV1Beta1IngressSpec(ingressToReuse.Spec),
				Strategy: k8gbv1beta1.Strategy{
					Type: strategy,
				},
			},
		}

		strToInt := func(str string) int {
			intValue, err := strconv.Atoi(str)
			if err != nil {
				log.Err(err).Msgf("can't convert string to int (%s)", err)
			}
			return intValue
		}

		for annotationKey, annotationValue := range a.GetAnnotations() {
			switch annotationKey {
			case dnsTTLSecondsAnnotation:
				gslb.Spec.Strategy.DNSTtlSeconds = strToInt(annotationValue)
			case splitBrainThresholdSecondsAnnotation:
				gslb.Spec.Strategy.SplitBrainThresholdSeconds = strToInt(annotationValue)
			}
		}

		if strategy == failoverStrategy {
			for annotationKey, annotationValue := range a.GetAnnotations() {
				if annotationKey == primaryGeoTagAnnotation {
					gslb.Spec.Strategy.PrimaryGeoTag = annotationValue
				}
			}
			if gslb.Spec.Strategy.PrimaryGeoTag == "" {
				log.Info().Msgf("%s annotation is missing, skipping Gslb creation...", primaryGeoTagAnnotation)
				return
			}
		}

		err = controllerutil.SetControllerReference(ingressToReuse, gslb, r.Scheme)
		if err != nil {
			log.Err(err).
				Str("Ingress", ingressToReuse.Name).
				Str("Gslb", gslb.Name).
				Msg("Cannot set the Ingress as the owner of the Gslb")
		}

		log.Info().Msgf("Creating new Gslb(%s) out of Ingress annotation", gslb.Name)
		err = c.Create(context.Background(), gslb)
		if err != nil {
			log.Err(err).Msg("Glsb creation failed")
		}
	}

	ingressMapHandler := handler.EnqueueRequestsFromMapFunc(
		func(a client.Object) []reconcile.Request {
			for annotationKey, annotationValue := range a.GetAnnotations() {
				if annotationKey == strategyAnnotation {
					switch annotationValue {
					case roundRobinStrategy:
						createGslbFromIngress(annotationKey, annotationKey, a, roundRobinStrategy)
					case failoverStrategy:
						createGslbFromIngress(annotationKey, annotationKey, a, failoverStrategy)
					}
				}
			}
			return nil
		})

	return ctrl.NewControllerManagedBy(mgr).
		For(&k8gbv1beta1.Gslb{}).
		Owns(&v1beta1.Ingress{}).
		Owns(&externaldns.DNSEndpoint{}).
		Watches(&source.Kind{Type: &corev1.Endpoints{}}, endpointMapHandler).
		Watches(&source.Kind{Type: &v1beta1.Ingress{}}, ingressMapHandler).
		Complete(r)
}
