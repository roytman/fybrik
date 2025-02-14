// Copyright 2021 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"testing"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"fybrik.io/fybrik/pkg/multicluster"
)

var _ multicluster.ClusterManager = &localClusterManager{}

func TestLocalClusterManager(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	s := scheme.Scheme
	objs := []runtime.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-metadata",
				Namespace: "fybrik-system",
			},
			Data: map[string]string{
				"ClusterName": "remote-cluster",
				"Region":      "Region-1",
				"Zone":        "Zone-1",
			},
		},
	}
	cl := fake.NewFakeClientWithScheme(s, objs...)
	namespace := "fybrik-system"
	cm, err := NewClusterManager(cl, namespace)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(cm).NotTo(gomega.BeNil())
	var actualClusters []multicluster.Cluster
	if actualClusters, err = cm.GetClusters(); err != nil {
		t.Errorf("unexpected error in GetClusters: %v", err)
	}

	expectedClusters := []multicluster.Cluster{
		{
			Name: "remote-cluster",
			Metadata: multicluster.ClusterMetadata{
				Region: "Region-1",
				Zone:   "Zone-1",
			},
		},
	}
	g.Expect(expectedClusters).To(gomega.Equal(actualClusters))
}
