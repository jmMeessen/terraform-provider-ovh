package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPublicCloudProjectKubeDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckKubernetes(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPublicCloudProjectKubeDatasourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccPublicCloudProjectKubeDatasource("data.ovh_cloud_project_kube.cluster"),
				),
			},
		},
	})
}

func testAccPublicCloudProjectKubeDatasource(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("can't find expected data: %s", n)
		}

		return nil
	}
}

var testAccPublicCloudProjectKubeDatasourceConfig = fmt.Sprintf(`
data "ovh_cloud_project_kube" "cluster" {
  project_id = "%s"
  name = "%s"
}
`, os.Getenv("OVH_PUBLIC_CLOUD"), os.Getenv("OVH_KUBERNETES_CLUSTER_NAME"))
