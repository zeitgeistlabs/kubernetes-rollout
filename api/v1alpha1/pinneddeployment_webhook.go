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

package v1alpha1

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var pinneddeploymentlog = logf.Log.WithName("pinneddeployment-resource")

func (r *PinnedDeployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-rollout-zeitgeistlabs-io-v1alpha1-pinneddeployment,mutating=true,failurePolicy=fail,groups=rollout.zeitgeistlabs.io,resources=pinneddeployments,verbs=create;update,versions=v1alpha1,name=mpinneddeployment.kb.io

var _ webhook.Defaulter = &PinnedDeployment{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PinnedDeployment) Default() {
	pinneddeploymentlog.Info("default", "name", r.Name)
	pinneddeploymentlog.Info("original", "object", r)

	if r.Spec.ReplicasRoundingStrategy == "" {
		r.Spec.ReplicasRoundingStrategy = NearestReplicasRoundingStrategyType
	}

	// We don't attempt to set PodSpec defaults in the templates.
	pinneddeploymentlog.Info("defaulted", "object", r)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-rollout-zeitgeistlabs-io-v1alpha1-pinneddeployment,mutating=false,failurePolicy=fail,groups=rollout.zeitgeistlabs.io,resources=pinneddeployments,versions=v1alpha1,name=vpinneddeployment.kb.io

var _ webhook.Validator = &PinnedDeployment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PinnedDeployment) ValidateCreate() error {
	pinneddeploymentlog.Info("validate create", "name", r.Name)

	return r.ValidateCreateUpdate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PinnedDeployment) ValidateUpdate(old runtime.Object) error {
	pinneddeploymentlog.Info("validate update", "name", r.Name)

	return r.ValidateCreateUpdate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PinnedDeployment) ValidateDelete() error {
	pinneddeploymentlog.Info("validate delete", "name", r.Name)

	return nil
}

// Common validation logic for creation and deletion.
func (r *PinnedDeployment) ValidateCreateUpdate() error {
	if r.Spec.ReplicasPercentNext < 0 || r.Spec.ReplicasPercentNext > 100 {
		return errors.New("ReplicasPercentNext must be between 0 and 100 (inclusive)")
	}

	// Handle enum checking.
	if !(r.Spec.ReplicasRoundingStrategy == DownReplicasRoundingStrategyType ||
		r.Spec.ReplicasRoundingStrategy == UpReplicasRoundingStrategyType ||
		r.Spec.ReplicasRoundingStrategy == NearestReplicasRoundingStrategyType) {
		return errors.New("ReplicasRoundingStrategy must be one of Up, Down, Nearest")
	}

	return nil
}
