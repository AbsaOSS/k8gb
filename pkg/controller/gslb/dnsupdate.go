package gslb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ohmyglbv1beta1 "github.com/AbsaOSS/ohmyglb/pkg/apis/ohmyglb/v1beta1"
	externaldns "github.com/kubernetes-incubator/external-dns/endpoint"
	"github.com/miekg/dns"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileGslb) getGslbIngressIPs(gslb *ohmyglbv1beta1.Gslb) ([]string, error) {
	nn := types.NamespacedName{
		Name:      gslb.Name,
		Namespace: gslb.Namespace,
	}

	gslbIngress := &v1beta1.Ingress{}

	err := r.client.Get(context.TODO(), nn, gslbIngress)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Can't find gslb Ingress: %s", gslb.Name))
		}
		return nil, err
	}

	var gslbIngressIPs []string

	for _, ip := range gslbIngress.Status.LoadBalancer.Ingress {
		gslbIngressIPs = append(gslbIngressIPs, ip.IP)
	}

	return gslbIngressIPs, nil
}

func (r *ReconcileGslb) getExternalTargets(gslb *ohmyglbv1beta1.Gslb) ([]string, error) {

	extGslbClustersVar := os.Getenv("EXT_GSLB_CLUSTERS")

	if extGslbClustersVar == "" {
		log.Info("No other Gslb enabled clusters are defined in the configuration...Working standalone")
		return nil, nil
	}

	extGslbClusters := strings.Split(extGslbClustersVar, ",")

	var targets []string

	for _, cluster := range extGslbClusters {
		log.Info(fmt.Sprintf("Adding external Gslb targets from %s cluster...", cluster))
		g := new(dns.Msg)
		hostsz := fmt.Sprintf("hostsz.%s.%s.", gslb.Name, cluster)
		g.SetQuestion(hostsz, dns.TypeA)
		ns := fmt.Sprintf("%s:53", cluster)
		a, err := dns.Exchange(g, ns)
		if err != nil {
			return nil, err
		}
		var clusterTargets []string

		for _, A := range a.Answer {
			IP := strings.Split(A.String(), "\t")[4]
			clusterTargets = append(clusterTargets, IP)
		}
		if len(clusterTargets) > 0 {
			targets = append(targets, clusterTargets...)
			log.Info(fmt.Sprintf("Added external %s Gslb targets from %s cluster", clusterTargets, hostsz))
		}
	}

	return targets, nil
}

func (r *ReconcileGslb) gslbDNSEndpoint(gslb *ohmyglbv1beta1.Gslb) (*externaldns.DNSEndpoint, error) {
	var gslbHosts []*externaldns.Endpoint

	serviceHealth, err := r.getServiceHealthStatus(gslb)
	if err != nil {
		return nil, err
	}

	targets, err := r.getGslbIngressIPs(gslb)
	if err != nil {
		return nil, err
	}

	dnsZone := os.Getenv("DNS_ZONE")

	// Service TXT DNS Record to share healthy target with ext. Gslb
	serviceDNSRecord := &externaldns.Endpoint{
		DNSName:    fmt.Sprintf("hostsz.%s.%s", gslb.Name, dnsZone),
		RecordTTL:  30,
		RecordType: "A",
		Targets:    targets,
	}
	gslbHosts = append(gslbHosts, serviceDNSRecord)

	externalTargets, err := r.getExternalTargets(gslb)
	if err != nil {
		return nil, err
	}

	if len(externalTargets) > 0 {
		switch gslb.Spec.Strategy {
		case "roundRobin":
			targets = append(targets, externalTargets...)
		}
	}

	for host, health := range serviceHealth {
		if health == "Healthy" {
			dnsRecord := &externaldns.Endpoint{
				DNSName:    host,
				RecordTTL:  30,
				RecordType: "A",
				Targets:    targets,
			}
			gslbHosts = append(gslbHosts, dnsRecord)
		}
	}
	dnsEndpointSpec := externaldns.DNSEndpointSpec{
		Endpoints: gslbHosts,
	}

	dnsEndpoint := &externaldns.DNSEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gslb.Name,
			Namespace: gslb.Namespace,
		},
		Spec: dnsEndpointSpec,
	}

	err = controllerutil.SetControllerReference(gslb, dnsEndpoint, r.scheme)
	if err != nil {
		return nil, err
	}
	return dnsEndpoint, err
}

func (r *ReconcileGslb) ensureDNSEndpoint(request reconcile.Request,
	instance *ohmyglbv1beta1.Gslb,
	i *externaldns.DNSEndpoint,
) (*reconcile.Result, error) {
	found := &externaldns.DNSEndpoint{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the DNSEndpoint
		log.Info(fmt.Sprintf("Creating a new DNSEndpoint:\n %s", prettyPrint(i)))
		err = r.client.Create(context.TODO(), i)

		if err != nil {
			// Creation failed
			log.Error(err, "Failed to create new DNSEndpoint", "DNSEndpoint.Namespace", i.Namespace, "DNSEndpoint.Name", i.Name)
			return &reconcile.Result{}, err
		}
		// Creation was successful
		return nil, nil
	} else if err != nil {
		// Error that isn't due to the service not existing
		log.Error(err, "Failed to get DNSEndpoint")
		return &reconcile.Result{}, err
	}

	// Update existing object with new spec
	found.Spec = i.Spec
	err = r.client.Update(context.TODO(), found)

	if err != nil {
		// Update failed
		log.Error(err, "Failed to update DNSEndpoint", "DNSEndpoint.Namespace", found.Namespace, "DNSEndpoint.Name", found.Name)
		return &reconcile.Result{}, err
	}

	return nil, nil
}

func prettyPrint(s interface{}) string {
	prettyStruct, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		fmt.Println("can't convert struct to json")
	}
	return string(prettyStruct)
}
