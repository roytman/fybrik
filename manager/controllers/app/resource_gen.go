// Copyright 2020 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"

	"emperror.dev/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	app "github.com/ibm/the-mesh-for-data/manager/apis/app/v1alpha1"
	"github.com/ibm/the-mesh-for-data/manager/controllers/utils"
	corev1 "k8s.io/api/core/v1"
)

// ContextInterface is an interface for communication with a generated resource (e.g. Blueprint)
type ContextInterface interface {
	ResourceExists(ref *app.ResourceReference) bool
	CreateOrUpdateResource(owner *app.ResourceReference, ref *app.ResourceReference, blueprintPerClusterMap map[string]app.BlueprintSpec) error
	DeleteResource(ref *app.ResourceReference) error
	GetResourceStatus(ref *app.ResourceReference) (app.ObservedState, error)
	CreateResourceReference(appName string, appNamespace string) (*app.ResourceReference, error)
	GetManagedObject() runtime.Object
}

// Interface for managing Blueprint resources

// BlueprintInterface context implementation for communication with a single Blueprint resource
type BlueprintInterface struct {
	Client client.Client
}

// GetManagedObject returns the type of the managed runtime object
func (c *BlueprintInterface) GetManagedObject() runtime.Object {
	return &app.Blueprint{}
}

// CreateResourceReference returns reference (name and namespace) to the generated resource.
// It also creates a namespace where the blueprint and the orchestrated resources will be running
func (c *BlueprintInterface) CreateResourceReference(appName string, appNamespace string) (*app.ResourceReference, error) {
	genNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "m4d-"}}
	genNamespace.Labels = ownerLabels(types.NamespacedName{Namespace: appNamespace, Name: appName})
	if err := c.Client.Create(context.Background(), genNamespace); err != nil {
		return nil, err
	}
	return &app.ResourceReference{Name: appName, Namespace: genNamespace.Name, Kind: "Blueprint"}, nil
}

// ResourceExists checks whether the blueprint resource generated by M4DApplication controller is active
func (c *BlueprintInterface) ResourceExists(ref *app.ResourceReference) bool {
	if ref == nil || ref.Namespace == "" {
		return false
	}
	resource := c.GetResourceSignature(ref)
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, resource); err != nil {
		return false
	}
	return true
}

// GetResourceSignature returns the namespaced information of the generated Blueprint resource
func (c *BlueprintInterface) GetResourceSignature(ref *app.ResourceReference) *app.Blueprint {
	return &app.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ref.Name,
			Namespace: ref.Namespace,
		},
	}
}

// CreateOrUpdateResource creates a new Blueprint resource or updates an existing one
func (c *BlueprintInterface) CreateOrUpdateResource(owner *app.ResourceReference, ref *app.ResourceReference, blueprintPerClusterMap map[string]app.BlueprintSpec) error {
	resource := c.GetResourceSignature(ref)
	if len(blueprintPerClusterMap) != 1 {
		return errors.New("Invalid cluster configuration")
	}
	// There is no actual iteration loop, since the map includes a single BlueprintSpec element
	for _, blueprintSpec := range blueprintPerClusterMap {
		if _, err := ctrl.CreateOrUpdate(context.Background(), c.Client, resource, func() error {
			resource.Spec = blueprintSpec
			resource.Labels = ownerLabels(types.NamespacedName{Namespace: owner.Namespace, Name: owner.Name})
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// DeleteResource deletes the generated Blueprint resource
func (c *BlueprintInterface) DeleteResource(ref *app.ResourceReference) error {
	resource := c.GetResourceSignature(ref)
	if err := c.Client.Delete(context.Background(), resource); err != nil {
		return err
	}
	return c.Client.Delete(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ref.Namespace,
		}})
}

// GetResourceStatus returns the generated Blueprint status
func (c *BlueprintInterface) GetResourceStatus(ref *app.ResourceReference) (app.ObservedState, error) {
	if ref == nil || ref.Namespace == "" {
		return app.ObservedState{}, nil
	}
	resource := c.GetResourceSignature(ref)
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, resource); err != nil {
		return app.ObservedState{}, err
	}
	return resource.Status.ObservedState, nil
}

// NewBlueprintInterface creates a new blueprint interface for M4DApplication controller
func NewBlueprintInterface(cl client.Client) *BlueprintInterface {
	return &BlueprintInterface{
		Client: cl,
	}
}

// Interface for managing Plotter resources

// PlotterInterface context implementation for communication with a single Plotter resource
type PlotterInterface struct {
	Client client.Client
}

// GetManagedObject returns the type of the managed runtime object
func (c *PlotterInterface) GetManagedObject() runtime.Object {
	return &app.Plotter{}
}

// CreateResourceReference returns an identifier (name and namespace) of the generated resource.
func (c *PlotterInterface) CreateResourceReference(appName string, appNamespace string) (*app.ResourceReference, error) {
	// Plotter runs in the control plane namespace. Plotter name identifies m4dapplication (name and namespace)
	return &app.ResourceReference{Name: appName + "-" + appNamespace, Namespace: utils.GetSystemNamespace(), Kind: "Plotter"}, nil
}

// ResourceExists checks whether the Plotter resource generated by M4DApplication controller is active
func (c *PlotterInterface) ResourceExists(ref *app.ResourceReference) bool {
	if ref == nil || ref.Namespace == "" {
		return false
	}
	resource := c.GetResourceSignature(ref)
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, resource); err != nil {
		return false
	}
	return true
}

// GetResourceSignature returns the namespaced information of the generated Plotter resource
func (c *PlotterInterface) GetResourceSignature(ref *app.ResourceReference) *app.Plotter {
	return &app.Plotter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ref.Name,
			Namespace: ref.Namespace,
		},
	}
}

// CreateOrUpdateResource creates a new Plotter resource or updates an existing one
func (c *PlotterInterface) CreateOrUpdateResource(owner *app.ResourceReference, ref *app.ResourceReference, blueprintPerClusterMap map[string]app.BlueprintSpec) error {
	resource := c.GetResourceSignature(ref)
	if len(blueprintPerClusterMap) == 0 {
		return errors.New("Invalid cluster configuration")
	}
	if _, err := ctrl.CreateOrUpdate(context.Background(), c.Client, resource, func() error {
		resource.Spec.Blueprints = blueprintPerClusterMap
		resource.Labels = ownerLabels(types.NamespacedName{Namespace: owner.Namespace, Name: owner.Name})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// DeleteResource deletes the generated Plotter resource
func (c *PlotterInterface) DeleteResource(ref *app.ResourceReference) error {
	resource := c.GetResourceSignature(ref)
	if err := c.Client.Delete(context.Background(), resource); err != nil {
		return err
	}
	return nil
}

// GetResourceStatus returns the generated Plotter status
func (c *PlotterInterface) GetResourceStatus(ref *app.ResourceReference) (app.ObservedState, error) {
	if ref == nil || ref.Namespace == "" {
		return app.ObservedState{}, nil
	}
	resource := c.GetResourceSignature(ref)
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, resource); err != nil {
		return app.ObservedState{}, err
	}
	return resource.Status.ObservedState, nil
}

// NewPlotterInterface creates a new plotter interface for M4DApplication controller
func NewPlotterInterface(cl client.Client) *PlotterInterface {
	return &PlotterInterface{
		Client: cl,
	}
}