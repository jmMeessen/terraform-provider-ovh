package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccCloudProjectKubeDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckKubernetes(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudProjectKubeDatasourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCloudProjectKubeDatasource("data.ovh_cloud_project_kube.cluster"),
				),
			},
		},
	})
}

func testAccCloudProjectKubeDatasource(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("can't find expected data: %s", n)
		}

		return nil
	}
}

var testAccCloudProjectKubeDatasourceConfig = fmt.Sprintf(`
data "ovh_cloud_project_kube" "cluster" {
  service_name = "%s"
  id = "%s"
}
`, os.Getenv("OVH_CLOUD_PROJECT_SERVICE"), os.Getenv("OVH_CLOUD_PROJECT_KUBE_SERVICE_TEST"))
