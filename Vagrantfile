# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  # config.vm.box = "ubuntu/focal64"     # Ubuntu 20.04 Focal Fossa (non CO-RE)
  # config.vm.box = "ubuntu/hirsute64"   # Ubuntu 21.04 Hirsute Hippo (CO-RE)
  config.vm.box = "ubuntu/impish64"      # Ubuntu 21.10 Impish Indri (CO-RE)

  # Forward MkDocs dev server to preview documentation on the host at http://localhost:8000/tracee
  config.vm.network :forwarded_port, guest: 8000, host: 8000

  # Forward MicroK8s dashboard to access it on the host at https://localhost:10443
  #
  # To access the Kubernetes dashboard from the host run the following command:
  #     kubectl port-forward --address 0.0.0.0 -n kube-system service/kubernetes-dashboard 10443:443
  #
  # To sing in use the token retrieved with
  #     token=$(microk8s kubectl -n kube-system get secret | grep default-token | cut -d " " -f1)
  #     kubectl -n kube-system describe secret $token
  #
  # TIP For Google Chrome you may allow insecure TLS connections at chrome://flags/#allow-insecure-localhost
  config.vm.network :forwarded_port, guest: 10443, host: 10443

  config.vm.provider "virtualbox" do |vb|
    vb.gui = false
    vb.memory = "2048"
  end

  config.vm.provision "shell", inline: <<-SHELL
    GO_VERSION="1.17"
    OPA_VERSION="v0.35.0"

    apt-get update
    apt-get install --yes build-essential pkgconf libelf-dev llvm-12 clang-12

    for tool in "clang" "llc" "llvm-strip"
    do
      path=$(which $tool-12)
      sudo ln -s $path ${path%-*}
    done

    snap install microk8s --classic
    microk8s status --wait-ready
    usermod -a -G microk8s vagrant
    microk8s enable dashboard

    mkdir -p /home/vagrant/.kube/
    microk8s kubectl config view --raw > /home/vagrant/.kube/config

    apt-get install --yes apt-transport-https ca-certificates curl
    curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg
    echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list
    apt-get update
    apt-get install --yes kubectl
    echo 'source <(kubectl completion bash)' >> /home/vagrant/.bashrc

    apt-get install --yes linux-tools-$(uname -r)

    apt-get install --yes docker.io
    usermod -aG docker vagrant

    wget --quiet https://golang.org/dl/go$GO_VERSION.linux-amd64.tar.gz
    tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /home/vagrant/.profile

    curl -L -o /usr/bin/opa https://github.com/open-policy-agent/opa/releases/download/$OPA_VERSION/opa_linux_amd64
    chmod 755 /usr/bin/opa
  SHELL
end
