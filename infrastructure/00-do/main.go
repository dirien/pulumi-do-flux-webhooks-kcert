package main

import (
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a DigitalOcean resource (Domain)
		kubernetesCluster, err := digitalocean.NewKubernetesCluster(ctx, "pulumi-do-flux-webhooks-kcert", &digitalocean.KubernetesClusterArgs{
			Name:        pulumi.String("pulumi-do-flux-webhooks-kcert"),
			Ha:          pulumi.Bool(false),
			Version:     pulumi.String("1.22.8-do.1"),
			AutoUpgrade: pulumi.Bool(false),
			MaintenancePolicy: digitalocean.KubernetesClusterMaintenancePolicyArgs{
				Day:       pulumi.String("sunday"),
				StartTime: pulumi.String("03:00"),
			},
			Region: pulumi.String("fra1"),
			NodePool: digitalocean.KubernetesClusterNodePoolArgs{
				Name:      pulumi.String("default"),
				Size:      pulumi.String("s-4vcpu-8gb"),
				AutoScale: pulumi.Bool(false),
				NodeCount: pulumi.Int(3),
			},
		})
		if err != nil {
			return err
		}

		bucket, err := digitalocean.NewSpacesBucket(ctx, "flux-bucket", &digitalocean.SpacesBucketArgs{
			Name:   pulumi.String("flux-bucket"),
			Region: pulumi.String("fra1"),
		})
		if err != nil {
			return err
		}

		doConfig := config.New(ctx, "digitalocean")

		ctx.Export("kubeconfig", pulumi.ToSecret(kubernetesCluster.KubeConfigs.ToKubernetesClusterKubeConfigArrayOutput().Index(pulumi.Int(0)).RawConfig()))
		ctx.Export("spaces_access_id", doConfig.GetSecret("spaces_access_id"))
		ctx.Export("spaces_secret_key", doConfig.GetSecret("spaces_secret_key"))
		ctx.Export("bucket", bucket.Name)
		ctx.Export("bucket-region", bucket.Region)

		return nil
	})
}
