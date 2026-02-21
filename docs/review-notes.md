# Review Notes: Helm Chart Update for Backend Storage Support

- Introduced backend storage configuration in `values.yaml`: `storage.enabled`, `storage.type` (`pvc` or `emptyDir`), and `storage.size`.
- Added `backend-pvc.yaml` template to create a PersistentVolumeClaim when storage is enabled with type `pvc`.
- Modified `backend-deployment.yaml` to conditionally mount the storage volume `/data` when storage is enabled.
- Deployment, service, and storage resource labels follow Kubernetes recommended conventions consistently.
- Readiness and liveness probes for the backend are properly defined and consistent.
- Changes respect DoD requirements: formatting, linting, testing, and build passed; no secrets introduced.
- No changes required; the implementation aligns well with existing Helm chart structure and repository standards.
