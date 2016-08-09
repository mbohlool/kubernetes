.PHONY:	build push

ETCD_VERSION = 2.2.1

clean:
	rm -rf backup etcdctl etcd-v$(ETCD_VERSION)-linux-amd64 etcd-v$(ETCD_VERSION)-linux-amd64.tar.gz

build: clean
	curl -L -O https://github.com/coreos/etcd/releases/download/v$(ETCD_VERSION)/etcd-v$(ETCD_VERSION)-linux-amd64.tar.gz
	tar xzvf etcd-v$(ETCD_VERSION)-linux-amd64.tar.gz
	cp etcd-v$(ETCD_VERSION)-linux-amd64/etcdctl .
	docker build -t mbohlool/etcd_cleanup .
	rm -rf backup etcdctl etcd-v$(ETCD_VERSION)-linux-amd64 etcd-v$(ETCD_VERSION)-linux-amd64.tar.gz
