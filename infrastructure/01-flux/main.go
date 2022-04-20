package main

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		cloud, err := pulumi.NewStackReference(ctx, "dirien/00-do/dev", nil)
		if err != nil {
			return err
		}

		provider, err := kubernetes.NewProvider(ctx, "kubernetes", &kubernetes.ProviderArgs{
			Kubeconfig: cloud.GetStringOutput(pulumi.String("kubeconfig")),
		})
		if err != nil {
			return err
		}

		fluxNS, err := v1.NewNamespace(ctx, "flux-system", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.StringPtr("flux-system"),
			},
		}, pulumi.Provider(provider), pulumi.Parent(provider))
		if err != nil {
			return err
		}

		flux, err := helm.NewRelease(ctx, "flux2", &helm.ReleaseArgs{
			Name:            pulumi.String("flux2"),
			Chart:           pulumi.String("flux2"),
			Version:         pulumi.String("0.16.0"),
			Namespace:       fluxNS.Metadata.Name(),
			CreateNamespace: pulumi.Bool(true),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://fluxcd-community.github.io/helm-charts"),
			},
			ValueYamlFiles: pulumi.AssetOrArchiveArray{
				pulumi.NewFileAsset("values/flux2.yaml"),
			},
		}, pulumi.Provider(provider), pulumi.Parent(fluxNS))
		if err != nil {
			return err
		}

		secret, err := v1.NewSecret(ctx, "flux-bucket-secret", &v1.SecretArgs{
			Metadata: metav1.ObjectMetaArgs{
				Name:      pulumi.StringPtr("flux-bucket-secret"),
				Namespace: fluxNS.Metadata.Name(),
			},
			StringData: pulumi.StringMap{
				"accesskey": cloud.GetStringOutput(pulumi.String("spaces_access_id")),
				"secretkey": cloud.GetStringOutput(pulumi.String("spaces_secret_key")),
			},
			Type: pulumi.String("Opaque"),
		}, pulumi.Provider(provider), pulumi.Parent(flux))
		if err != nil {
			return err
		}

		bucketCR, err := apiextensions.NewCustomResource(ctx, "do-bucket", &apiextensions.CustomResourceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("do-bucket"),
				Namespace: fluxNS.Metadata.Name(),
			},
			ApiVersion: pulumi.String("source.toolkit.fluxcd.io/v1beta2"),
			Kind:       pulumi.String("Bucket"),
			OtherFields: kubernetes.UntypedArgs{
				"spec": &pulumi.Map{
					"interval":   pulumi.String("1m0s"),
					"provider":   pulumi.String("generic"),
					"bucketName": cloud.GetStringOutput(pulumi.String("bucket")),
					"endpoint":   pulumi.String("fra1.digitaloceanspaces.com"),
					"region":     cloud.GetStringOutput(pulumi.String("bucket-region")),
					"secretRef": &pulumi.Map{
						"name": secret.Metadata.Name(),
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.Parent(flux))
		if err != nil {
			return err
		}
		_, err = apiextensions.NewCustomResource(ctx, "demo-kustomization", &apiextensions.CustomResourceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("demo-kustomization"),
				Namespace: fluxNS.Metadata.Name(),
			},
			ApiVersion: pulumi.String("kustomize.toolkit.fluxcd.io/v1beta2"),
			Kind:       pulumi.String("Kustomization"),
			OtherFields: kubernetes.UntypedArgs{
				"spec": &pulumi.Map{
					"interval": pulumi.String("1m0s"),
					"path":     pulumi.String("./"),
					"prune":    pulumi.Bool(true),
					"sourceRef": &pulumi.Map{
						"kind":      bucketCR.Kind.ToStringOutput(),
						"name":      bucketCR.Metadata.Name(),
						"namespace": fluxNS.Metadata.Name(),
					},
				},
			},
		}, pulumi.Provider(provider), pulumi.Parent(bucketCR))
		if err != nil {
			return err
		}

		return nil
	})
}
