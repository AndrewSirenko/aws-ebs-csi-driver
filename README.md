# Simplified aws-ebs-csi-driver

## Codewalk notes

Trimmed aws-ebs-csi-driver file-tree

```
.
├── Dockerfile
├── Makefile
├── go.mod
│
├── cmd
│
└── pkg
   │
   ├── cloud
   │   └── cloud.go
   │
   ├── driver
   │   ├── controller.go
   │   ├── driver.go
   │   ├── identity.go
   │   ├── node.go
   │   └── options.go
   │
   ├── mounter
   │
   └── util
```


/go.mod: Declares our project dependencies, versions and module path.

/cmd: Contains project's executable entry-points (`main` functions). It's convention to separate binary from application (See appendix).

/cmd/main.go: Parses command-line flags, glues together dependencies, sets up an EBS CSI Driver, "Runs" the driver.

/pkg: Contains all application packages

/pkg/driver: Implements CSI Spec

/pkg/driver/driver.go: Sets up driver in controller or node mode, starts listening on gRPC endpoint, and serves CSI Controller OR Node Service RPC requests.

/pkg/driver/controller.go: Implements Controller Service RPCs of CSI Spec. Should offload AWS EC2 specific logic to cloud package.

/pkg/driver/node.go: Implements Node Service RPCs of CSI Spec. Should offload OS specific logic to mounter package.

/pkg/driver/identity: Implements Identity Service RPC.

pkg/cloud: Package for all cloud-provider interface and EC2-specific implementation. Driver package shouldn't need to know about EC2 APIs.

pkg/mounter: Mounter interface and implementations so driver package doesn't need to know OS mount/formatting syscalls for different platforms. I.e. Node.go doesn't know what Linux or Windows is.

`make` builds aws-ebs-csi-driver binary (in /bin) -> containerized with Docker

---

What happens when EBS CSI Driver deployed:

EBS CSI Driver helm chart deploys `aws-ebs-csi-driver` container as `ebs-plugin` container within ebs-csi-controller Deployment and ebs-csi-node Daemonset.

On `ebs-plugin` container startup: cmd/main.go main function instantiates various interface implementations, and runs driver from pkg/driver/driver.go

`ebs-plugin` now listens for RPCs on the CSI Endpoint, and will spawn a new go-routine to handle each call

In the ebs-csi-node Daemonset:

Kubelet can now call Node CSI RPCs (like NodeStageVolume), which will execute driver code from the relevant function in pkg/driver/node.go

In ebs-csi-controller Deployment:

Sidecars like csi-provisioner can call Controller CSI RPCs (like ControllerCreateVolume), which will execute driver code from relevant function in pkg/driver/controller.go



## Diagrams

<img width="1048" alt="image" src="https://github.com/user-attachments/assets/d5c6366b-9adf-4a64-b8ac-9eade35e884d" />
<img width="1051" alt="image" src="https://github.com/user-attachments/assets/f4a6b2f7-85cd-41b6-b80b-ae26af1e0a63" />
<img width="1148" alt="image" src="https://github.com/user-attachments/assets/13d0a421-b2fa-4b61-a675-a320ae77f74a" />
<img width="1076" alt="image" src="https://github.com/user-attachments/assets/61bbf0f7-58f3-45d3-a281-1cb871f03852" />
<img width="822" alt="image" src="https://github.com/user-attachments/assets/8601b716-2a6f-46c9-bc75-5bdff1ee0c92" />
<img width="850" alt="image" src="https://github.com/user-attachments/assets/b95c7694-2328-478c-8f27-0c77ff48cf17" />
<img width="817" alt="image" src="https://github.com/user-attachments/assets/816d17d6-3880-40cd-ae30-c25efd5be2b7" />






