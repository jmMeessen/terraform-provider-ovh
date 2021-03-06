---
layout: "ovh"
page_title: "OVH: cloud_project_region"
sidebar_current: "docs-ovh-datasource-cloud-project-region-x"
description: |-
  Get information & status of a region associated with a public cloud project.
---

# ovh_cloud_project_region

Use this data source to retrieve information about a region associated with a
public cloud project. The region must be associated with the project.

## Example Usage

```hcl
data "ovh_cloud_project_region" "GRA1" {
   project_id = "XXXXXX"
   name = "GRA1"
}
```

## Argument Reference


* `project_id` - (Optional) Deprecated. The id of the public cloud project. If omitted,
    the `OVH_PROJECT_ID` environment variable is used.
    One of `service_name` or `project_id` is required. Conflits with `service_name`.

* `service_name` - (Optional) The id of the public cloud project. If omitted,
    the `OVH_CLOUD_PROJECT_SERVICE` environment variable is used. 
    One of `service_name` or `project_id` is required. Conflits with `project_id`.

* `name` - (Required) The name of the region associated with the public cloud
project.

## Attributes Reference

`id` is set to the ID of the project concatenated with the name of the region.
In addition, the following attributes are exported:

* `continent_code` - the code of the geographic continent the region is running.
E.g.: EU for Europe, US for America...
* `datacenter_location` - The location code of the datacenter.
E.g.: "GRA", meaning Gravelines, for region "GRA1"
* `continentCode` - (Deprecated) Use `continent_code` instead.
* `datacenterLocation` - (Deprecated) Use `datacenter_location` instead.
* `services` - The list of public cloud services running within the region
  * `name` - the name of the public cloud service
  * `status` - the status of the service
