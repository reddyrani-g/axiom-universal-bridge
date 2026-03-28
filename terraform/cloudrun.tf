resource "google_cloud_run_v2_service" "axiom_bridge" {
  name     = "axiom-bridge"
  location = "us-central1"

  template {
    containers {
      image = "gcr.io/your-project-id/axiom-bridge:latest" # CI/CD will override this
      ports {
        container_port = 8080
      }
      env {
        name  = "GOOGLE_CLOUD_LOCATION"
        value = "us-central1"
      }
    }
  }
}

resource "google_cloud_run_service_iam_binding" "public" {
  location = google_cloud_run_v2_service.axiom_bridge.location
  project  = google_cloud_run_v2_service.axiom_bridge.project
  service  = google_cloud_run_v2_service.axiom_bridge.name

  role    = "roles/run.invoker"
  members = [
    "allUsers"
  ]
}
