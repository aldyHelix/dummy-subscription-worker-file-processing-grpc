DIR=deployments/docker
DOCKER_ENGINE ?= podman

RECIPE=${DIR}/docker-compose.yaml
NAMESPACE=builder${COMPONENT}${CI_JOB_ID}
NEXUS_GCP_REPO=
NEXUS_ALI_REPO=

include .env
export $(shell sed 's/=.*//' .env)


DIND_PREFIX ?= $(HOME)
ifneq ($(HOST_PATH),)
DIND_PREFIX := $(HOST_PATH)
endif
ifeq ($(CACHE_PREFIX),)
	CACHE_PREFIX=/tmp
endif

ifeq ($(PATH_MAIN),) 
	 PATH_MAIN= $(PWD)
endif

PREFIX=$(shell echo $(PATH_MAIN) | sed -e s:$(HOME):$(DIND_PREFIX):)

IMAGE_TAG ?= master

export $IMAGE_TAG
UID=$(shell whoami)

export DIND_PATH=${PREFIX}

infratest: 
	docker network create -d bridge ${NAMESPACE}_default ; #/bin/true
	docker-compose -f ${RECIPE} -p ${NAMESPACE} pull || true 
	docker-compose -f $(RECIPE) -p $(NAMESPACE) up -d --force-recreate testdb

test-local: infratest quicktest

test-local-mac: reset-test-local-mac infratest quicktest

test: clean-db
	ls -lR ./migrations
	echo "prepare test"
	gcloud auth activate-service-account --key-file ${GCS_CREDENTIAL_PATH}
	gsutil cp ORD-TA-NW-UP-20210323000000-0001.zip gs://dummy-bucket-upload-jakarta/dummy/${FTO_LOCAL_DIRECTORY}/ORD-TA-NW-UP-20210323000000-0001.zip
	MIGRATION_PATH=./migrations/test LOCAL_TEST=0 go test -test.parallel 4 -cover -coverprofile=coverage.out
	go tool cover -func=coverage.out

quicktest: clean-db
	docker run \
		--network ${NAMESPACE}_default \
		--env-file .env \
		-e HTTP_PROXY \
		-e GOPROXY \
		-e DB_HOST=${TEST_DB_HOST} \
		-e DB_USER=${TEST_DB_USER} \
		-e DB_PORT=${TEST_DB_PORT} \
		-e DB_NAME=${TEST_DB_NAME} \
		-e DB_PASS=${TEST_DB_PASS} \
		-e LOCAL_TEST=1 \
		-v $(CACHE_PREFIX)/cache/go:/go/pkg/mod \
		-v $(CACHE_PREFIX)/cache/apk:/etc/apk/cache \
		-v $(PREFIX)/deployments/docker/build:/build \
		-v $(PREFIX)/:/src \
		-v $(PREFIX)/migrations:/migrations \
		-v `pwd`/certs/dummy.com.abc:/dummy.com.abc \
		-v $(PREFIX)/scripts/test.sh:/test.sh \
		-e UID=$(UID) \
		${CONTAINER_REPO}/module-builder /test.sh $(TESTARGS)

gen:
	docker run -v $(PREFIX):/gen:z -v $(PREFIX)/api:/api citradigital/toldata:v0.1.5-busless\
		-I /protobuf -I /api/ \
		/api/${COMPONENT}.proto \
		--toldata_out=busless,grpc,rest:/gen --gogofaster_out=plugins=grpc:/gen

build-module-builder:
	docker-compose -f ${RECIPE} -p ${NAMESPACE} build module-builder 

build-migrate:
	docker-compose -f ${RECIPE} -p ${NAMESPACE} build --no-cache migrate 

build-api:
	CGO_ENABLED=0 GOOS=linux go build -o deployments/docker/build/api ./cmd/api/main.go

create-api: clean-api build-api
	docker-compose -f ${RECIPE} -p ${NAMESPACE} build --no-cache api 

clean-test:
	docker-compose -f ${RECIPE} -p ${NAMESPACE} stop 
	docker-compose -f ${RECIPE} -p ${NAMESPACE} rm -f testdb

clean-api:
	docker-compose -f ${RECIPE} -p ${NAMESPACE} stop 
	docker-compose -f ${RECIPE} -p ${NAMESPACE} rm -f api

run-api: build-api
	docker run \
		--network ${NAMESPACE}_default \
		-p 8000:8000 \
		--env-file .env \
		-v `pwd`/deployments/docker/build/api:/api \
		-v `pwd`/templates:/templates \
		-v `pwd`/certs/dummy.com.abc:/dummy.com.abc \
		docker.a.cicit.dev/core/simply-test-base /api

migrate:
	docker run --network $(NAMESPACE)_default -v `pwd`/migrations/deploy:/migrations migrate/migrate -source file://migrations -database 'postgres://${TEST_DB_USER}:${TEST_DB_PASS}@testdb:5432/testdb?sslmode=disable' drop
	docker run --network $(NAMESPACE)_default -v `pwd`/migrations/deploy:/migrations migrate/migrate -source file://migrations -database 'postgres://${TEST_DB_USER}:${TEST_DB_PASS}@testdb:5432/testdb?sslmode=disable' up

clean-db: 
	# echo "DELETE FROM participants" | docker exec --user postgres -i $$(docker ps | grep ${NAMESPACE}_testdb_1 | cut -d ' ' -f 1) psql -U db testdb || true
	# echo "DELETE FROM address_item" | docker exec --user postgres -i $$(docker ps | grep ${NAMESPACE}_testdb_1 | cut -d ' ' -f 1) psql -U db testdb || true
	# echo "DELETE FROM bank_account" | docker exec --user postgres -i $$(docker ps | grep ${NAMESPACE}_testdb_1 | cut -d ' ' -f 1) psql -U db testdb || true
	# echo "DELETE FROM customer_info" | docker exec --user postgres -i $$(docker ps | grep ${NAMESPACE}_testdb_1 | cut -d ' ' -f 1) psql -U db testdb || true
	# echo "DELETE FROM personal_item" | docker exec --user postgres -i $$(docker ps | grep ${NAMESPACE}_testdb_1 | cut -d ' ' -f 1) psql -U db testdb || true
	# echo "DELETE FROM attempt" | docker exec --user postgres -i $$(docker ps | grep ${NAMESPACE}_testdb_1 | cut -d ' ' -f 1) psql -U db testdb || true
	# echo "DELETE FROM participants_simple" | docker exec --user postgres -i $$(docker ps | grep ${NAMESPACE}_testdb_1 | cut -d ' ' -f 1) psql -U db testdb || true

clean-grpc:
	docker-compose -f ${RECIPE} -p ${NAMESPACE} stop 
	docker-compose -f ${RECIPE} -p ${NAMESPACE} rm -f grpcapi

create-grpc: clean-api build-grpc
	docker-compose -f ${RECIPE} -p ${NAMESPACE} build --no-cache grpcapi

build-grpc:
	CGO_ENABLED=0 GOOS=linux go build -o ./deployments/docker/build/grpc ./cmd/grpc/main.go

run-grpc: build-grpc
	docker run \
		--network host\
		--env-file .env \
		-e LOCAL_TEST=1 \
		-v `pwd`/deployments/docker/build/grpc:/grpc \
		-v `pwd`/certs/bucket-upload-dev.json:/bucket-upload-dev.json \
		alpine /grpc

gen-only:
	docker run -v $(PREFIX):/gen -v $(PREFIX)/api:/api citradigital/toldata:v0.1.4 \
		-I /protobuf -I /api/ \
		/api/$(COMPONENT).proto \
		--toldata_out=busless,grpc:/gen --gogofaster_out=plugins=grpc:/gen

swagger: gen-only
	docker run -v --rm -it -v $(PREFIX):/work -w /work quay.io/goswagger/swagger generate spec -m -o swagger.json

push: 
	docker-compose -f ${RECIPE} -p ${NAMESPACE} push api
	docker-compose -f ${RECIPE} -p ${NAMESPACE} push grpcapi


deploy-kubernetes:
	cat kubernetes/gcp-deployment-template.yaml | sed "s/{COMPONENT}/${COMPONENT}/g"  \
	| sed "s~{CONTAINER_REPO}~${CONTAINER_REPO}~g" | sed "s/{IMG_TAG}/${IMAGE_TAG}/g" \
	| sed "s/{CPU_REQ}/${CPU_REQ}/g" | sed "s/{MEMORY_REQ}/${MEMORY_REQ}/g" \
	| sed "s/{CPU_LIMIT}/${CPU_LIMIT}/g" | sed "s/{MEMORY_LIMIT}/${MEMORY_LIMIT}/g" 
	
	cat kubernetes/gcp-deployment-template.yaml | sed "s/{COMPONENT}/${COMPONENT}/g"  \
	| sed "s~{CONTAINER_REPO}~${CONTAINER_REPO}~g" | sed "s/{IMG_TAG}/${IMAGE_TAG}/g" \
	| sed "s/{CPU_REQ}/${CPU_REQ}/g" | sed "s/{MEMORY_REQ}/${MEMORY_REQ}/g" \
	| sed "s/{CPU_LIMIT}/${CPU_LIMIT}/g" | sed "s/{MEMORY_LIMIT}/${MEMORY_LIMIT}/g" \
	| kubectl delete  --kubeconfig=/jenkinsdev01/jenkins-home/kubeconfig/config-gke-dev -n dummy -f - --ignore-not-found=true

	cat kubernetes/gcp-deployment-template.yaml | sed "s/{COMPONENT}/${COMPONENT}/g"  \
	| sed "s~{CONTAINER_REPO}~${CONTAINER_REPO}~g" | sed "s/{IMG_TAG}/${IMAGE_TAG}/g" \
	| sed "s/{CPU_REQ}/${CPU_REQ}/g" | sed "s/{MEMORY_REQ}/${MEMORY_REQ}/g" \
	| sed "s/{CPU_LIMIT}/${CPU_LIMIT}/g" | sed "s/{MEMORY_LIMIT}/${MEMORY_LIMIT}/g" \
	| kubectl apply  --kubeconfig=/jenkinsdev01/jenkins-home/kubeconfig/config-gke-dev -n dummy -f -

	cat kubernetes/gcp-services-template.yaml | sed "s/{COMPONENT}/${COMPONENT}/g"  \
	| kubectl apply  --kubeconfig=/jenkinsdev01/jenkins-home/kubeconfig/config-gke-dev -n dummy -f -
	
	kubectl rollout --kubeconfig=/jenkinsdev01/jenkins-home/kubeconfig/config-gke-dev -n dummy status deployment/dummy-${COMPONENT}-v1 
	kubectl rollout --kubeconfig=/jenkinsdev01/jenkins-home/kubeconfig/config-gke-dev -n dummy status deployment/dummy-${COMPONENT}-grpc-v1  

reset-test-local-mac:
	docker container stop $(NAMESPACE)_testdb_1
	docker network rm ${NAMESPACE}_default

clean-all:
	docker-compose -f ${RECIPE} -p ${NAMESPACE} down --rmi all

run-grpc-worker:
	go run cmd/grpc/main.go