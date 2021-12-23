%ifarch x86_64
%global gohostarch amd64
%endif
%ifarch aarch64
%global gohostarch arm64
%endif

%define debug_package %{nil}

Name: kubeedge
Version: 1.8.0
Release: 1
Summary: Kubernetes Native Edge Computing Framework
License: Apache-2.0
URL: https://github.com/kubeedge/kubeedge
Source0: https://github.com/kubeedge/kubeedge/archive/refs/tags/v%{version}.tar.gz
BuildRequires: golang glibc-static make tar systemd git
Requires: mosquitto

%description
KubeEdge is an open source system for extending native containerized application
orchestration capabilities to hosts at Edge. It is built upon kubernetes and provides
fundamental infrastructure support for network, app, deployment and metadata
synchronization between cloud and edge.

%package keadm
Summary: Keadm is used to install the cloud and edge components of KubeEdge.
Provides: keadm = %{version}

%description keadm
Keadm is used to install the cloud and edge components of KubeEdge.
It is not responsible for installing K8s and runtime.

%package cloudcore
Summary: KubeEdge Cloud Agent (CloudCore)
Provides: cloudcore = %{version}
Provides: admission = %{version}
Provides: csidriver = %{version}

%description cloudcore
KubeEdge is built upon Kubernetes and extends native containerized application
orchestration and device management to hosts at the Edge. It consists of cloud
part and edge part, provides core infrastructure support for networking,
application deployment and metadata synchronization between cloud and edge.
This package contains the cloudcore binary for the cloud node.

%package edgecore
Summary: KubeEdge Lightweight Edge Agent (EdgeCore)
Provides: edgecore = %{version}

%description edgecore
KubeEdge is built upon Kubernetes and extends native containerized application
orchestration and device management to hosts at the Edge. It consists of cloud
part and edge part, provides core infrastructure support for networking,
application deployment and metadata synchronization between cloud and edge.
This package contains the edgecore binary for the edge node.

%package edgesite
Summary: A GRPC agent/server
Provides: edgesite-agent = %{version}
Provides: edgesite-server = %{version}

%description edgesite
edgesite-agent is a gRPC agent. Connects to the proxy and then allows traffic to be forwarded to it.
edgesite-server is a gRPC proxy server, receives requests from the API server and forwards to the agent.

%prep
%autosetup -Sgit -n %{name}-%{version} -p1

%build
# add git tag since tarball did not contain any git info
git tag -a v%{version} -m "%{version}"
# setup GOPATH
export GOPATH=%{_builddir}
%global workspace $GOPATH/src/github.com/%{name}/%{name}
mkdir -p $GOPATH/src/github.com/%{name}
ln -sf $PWD $GOPATH/src/github.com/%{name}/%{name}
# start to build
cd %{workspace}
# set go flags
export GOLDFLAGS="-buildid=none -buildmode=pie -extldflags=-ftrapv -extldflags=-zrelro -extldflags=-znow -linkmode=external -extldflags=-static"
# build binaries
make all
# build csidriver
# TODO: delete after PR3154 is merged in new release(not included in v1.8.0)
go build -v -o _output/local/bin/csidriver -ldflags="${GOLDFLAGS}" github.com/kubeedge/kubeedge/cloud/cmd/csidriver

%install
export GOPATH=%{_builddir}
cd %{workspace}
# create directories
install -dm0755 %{buildroot}%{_sysconfdir}/kubeedge
install -dm0755 %{buildroot}%{_sysconfdir}/kubeedge/config
install -dm0755 %{buildroot}%{_sysconfdir}/kubeedge/tools
# install binaries
install -Dpm0755 ./_output/local/bin/keadm %{buildroot}%{_prefix}/local/bin/keadm
install -Dpm0755 ./_output/local/bin/cloudcore %{buildroot}%{_prefix}/local/bin/cloudcore
install -Dpm0755 ./_output/local/bin/edgecore %{buildroot}%{_prefix}/local/bin/edgecore
install -Dpm0755 ./_output/local/bin/admission %{buildroot}%{_prefix}/local/bin/admission
install -Dpm0755 ./_output/local/bin/csidriver %{buildroot}%{_prefix}/local/bin/csidriver
install -Dpm0755 ./_output/local/bin/edgesite-agent %{buildroot}%{_prefix}/local/bin/edgesite-agent
install -Dpm0755 ./_output/local/bin/edgesite-server %{buildroot}%{_prefix}/local/bin/edgesite-server
# generate default configs for both cloudcore and edgecore
./_output/local/bin/cloudcore --defaultconfig > cloudcore.example.yaml
./_output/local/bin/edgecore --defaultconfig > edgecore.example.yaml
install -Dpm0644 ./cloudcore.example.yaml %{buildroot}%{_sysconfdir}/kubeedge/config/cloudcore.example.yaml
install -Dpm0644 ./edgecore.example.yaml %{buildroot}%{_sysconfdir}/kubeedge/config/edgecore.example.yaml
# service file for systemd
install -Dpm0644 ./build/tools/cloudcore.service %{buildroot}%{_unitdir}/cloudcore.service
install -Dpm0644 ./build/tools/edgecore.service %{buildroot}%{_unitdir}/edgecore.service
# install service file in /etc/kubeedge/ as well so that no need to download from internet when they use keadm
install -Dpm0644 ./build/tools/cloudcore.service %{buildroot}%{_sysconfdir}/kubeedge/cloudcore.service
install -Dpm0644 ./build/tools/edgecore.service %{buildroot}%{_sysconfdir}/kubeedge/edgecore.service
# crd yamls for kubernetes
cd ./build && find ./crds -type f -exec install -Dm0644 {} %{buildroot}%{_sysconfdir}/kubeedge/{} \; && cd -
# tool for certificate generation
install -Dpm0755 ./build/tools/certgen.sh %{buildroot}%{_sysconfdir}/kubeedge/tools/certgen.sh
# construct tarball used for keadm
%global tarball_name %{name}-v%{version}-linux-%{gohostarch}
install -Dpm0755 ./_output/local/bin/cloudcore %{tarball_name}/cloud/cloudcore/cloudcore
install -Dpm0755 ./_output/local/bin/admission %{tarball_name}/cloud/admission/admission
install -Dpm0755 ./_output/local/bin/csidriver %{tarball_name}/cloud/csidriver/csidriver
install -Dpm0755 ./_output/local/bin/edgecore %{tarball_name}/edge/edgecore
# like cp -r, but have filemode control here
cd ./build && find ./crds -type f -exec install -Dpm0644 {} ../%{tarball_name}/{} \; && cd -
echo "v%{version}" > %{tarball_name}/version
tar zcf %{tarball_name}.tar.gz %{tarball_name}
# checksum for tarball
sha512sum %{tarball_name}.tar.gz | awk '{print $1}' > checksum_%{tarball_name}.tar.gz.txt
# install tarball
install -Dpm0644 %{tarball_name}.tar.gz %{buildroot}%{_sysconfdir}/kubeedge/%{tarball_name}.tar.gz
install -Dpm0644 checksum_%{tarball_name}.tar.gz.txt %{buildroot}%{_sysconfdir}/kubeedge/checksum_%{tarball_name}.tar.gz.txt

%files keadm
%license LICENSE
%{_prefix}/local/bin/keadm
%{_sysconfdir}/kubeedge/cloudcore.service
%{_sysconfdir}/kubeedge/edgecore.service
%{_sysconfdir}/kubeedge/%{tarball_name}.tar.gz
%{_sysconfdir}/kubeedge/checksum_%{tarball_name}.tar.gz.txt

%files cloudcore
%license LICENSE
%{_prefix}/local/bin/cloudcore
%{_prefix}/local/bin/admission
%{_prefix}/local/bin/csidriver
%{_unitdir}/cloudcore.service
%{_sysconfdir}/kubeedge/crds
%{_sysconfdir}/kubeedge/tools/certgen.sh
%config(noreplace) %{_sysconfdir}/kubeedge/config/cloudcore.example.yaml

%files edgecore
%license LICENSE
%{_prefix}/local/bin/edgecore
%{_unitdir}/edgecore.service
%config(noreplace) %{_sysconfdir}/kubeedge/config/edgecore.example.yaml

%files edgesite
%license LICENSE
%{_prefix}/local/bin/edgesite-agent
%{_prefix}/local/bin/edgesite-server

%changelog
* Thu Sep 09 2021 CooperLi<a710905118@163.com> - 1.8.0-1
- Package init
