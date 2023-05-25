// ::DEFINE
def image_name          = "abc-copy-file-dummy-order-grpc"
def service_name        = "abc-copy-file-dummy-order-grpc"
def repo_name           = "dummy/abc-file-dummy-order-grpc" 
def appName             = "abc-copy-file-dummy-order-grpc"
def unitTest_standard   = "0.0%"
def sonarSrc            = "v1"
def sonarTest           = "v1"
def sonarCoverage       = "coverage.out"

// ::CLUSTER
def cluster_dev         = ""      
def cluster_staging     = ""   
def cluster_prod        = ""
def credentials_dev     = ""
def credentials_staging = ""
def credentials_prod    = ""

// ::NOTIFICATION
def telegram_url        = "" 
def telegram_chatid     = ""
def job_success         = "SUCCESS"
def job_error           = "ERROR"

// ::URL
def repo_url            = "https://gitlab.dummy.com/${repo_name}.git"
def docker_dev          = ""
def docker_staging      = ""
def docker_prod         = ""
def docker_credentials  = "ci-cd"
def helm_git            = ""
def artifact_dev        = ""
def artifact_staging    = ""
def artifact_prod       = ""
def nexusPluginsRepoUrl = ""
def nexusGoCentral      = ""
def k6_repo             = ""
def katalonRepoUrl      = ""
def host                = ""
def burpsuite_url       = ""
def service_staging_url = ""
def spinnaker_webhook	 = ""

// ENDPOINTS
def endpoint_urls        = ["\"${service_staging_url}/\""]

// ::KATALON
def katalonProjectName  = "TestPlease.prj"
def katalonTestSuiteName= "TestSuiteDummy"

// ::INITIALIZATION
def fullname            = "${service_name}-${env.BUILD_NUMBER}"
def version, helm_dir, runPipeline, unitTest_score

environment {
    GO111MODULE = 'on'
}

node ('master') {
    try {
        // ::CHECKOUT
        stage ("Checkout") {
            // ::TRIGGER
            if (env.GET_TRIGGER == 'staging') {
                runPipeline = 'staging'
                runBranch   = "*/master"
            } else if (env.GET_TRIGGER == 'production') {
                runPipeline = 'production'
                runBranch   = "*/tags/release-*"
            } else {
                runPipeline = 'dev'
                runBranch   = "*/${env.BRANCH_NAME}"
            }

            echo "With branch ${env.BRANCH_NAME}"

            // ::SOURCE CHECKOUT
            def scm = checkout([$class: 'GitSCM', branches: [[name: runBranch]], userRemoteConfigs: [[credentialsId: 'ci-cd', url: repo_url]]])

            // ::VERSIONING
            if (runPipeline == 'dev' && scm.GIT_BRANCH == 'origin/dev') {
                echo "Running DEV Pipeline with ${scm.GIT_BRANCH} branch"
                version             = "alpha"
                helm_dir            = "test"
                serverUrl           = "${cluster_dev}"
                docker_url          = "${docker_dev}"
                artifact_url        = "${artifact_dev}"
                credentialsCluster  = "${credentials_dev}"
            } else if (runPipeline == 'staging') {
                echo "Running Staging Pipeline with ${scm.GIT_BRANCH} branch"
                version             = "beta"
                helm_dir            = "incubator"
                serverUrl           = "${cluster_staging}"
                docker_url          = "${docker_staging}"
                artifact_url        = "${artifact_staging}"
                credentialsCluster  = "${credentials_staging}"
            } else if (runPipeline == 'production') {
                echo "Running Production Pipeline with tag ${scm.GIT_BRANCH}"
                version             = "release"
                helm_dir            = "stable"
                serverUrl           = "${cluster_prod}"
                docker_url          = "${docker_prod}"
                artifact_url        = "${artifact_prod}"
                credentialsCluster  = "${credentials_prod}"
            } else {
                echo "Running DEV Pipeline with ${scm.GIT_BRANCH} branch"
                version             = "debug"
                helm_dir            = "test"
                serverUrl           = "${cluster_dev}"
                docker_url          = "${docker_dev}"
                artifact_url        = "${artifact_dev}"
                credentialsCluster  = "${credentials_dev}"
            }
        }
        // ::DEV-STAGING
        if (version != "release") {
            stage ("Unit Test") {
                echo "Running Unit Test"
                //sh "make test"
                    
                // goEnv()
    
                // def sts = 1
                // try {
                //     sts = sh (
                //         returnStatus: true, 
                //         script: '''
                //         export PATH=$PATH:$(go env GOPATH)/bin
                //         CGO_ENABLED=0 go test -test.parallel 4 -cover -v -covermode=count -coverprofile=coverage.out 2>&1 | go-junit-report -set-exit-code > ./report.xml

                //         echo $?
                //         '''
                //     )
                //     sh "go tool cover -func=coverage.out"
                //        echo sts.toString()
                // } catch (e) {
                //     echo "${e}"
                // }
    
                //finally{
                //    if (fileExists('./report.xml')) { 
                //        echo 'junit report'
                //        try{
                //            junit '**/report.xml'
                //        } catch(e) {
                //        }
                //    }
                //    if(sts == 1){
                //        error('Unit testing Fail!!!!')
                //    }
                //}
    
                // def unitTestGetValue = sh(returnStdout: true, script: 'go tool cover -func=coverage.out | grep total | sed "s/[[:blank:]]*$//;s/.*[[:blank:]]//"')
                // unitTest_score = "Your score is ${unitTestGetValue}"
                // echo "${unitTest_score}"
                // if (unitTestGetValue >= unitTest_standard) {
                //     echo "Unit Test fulfill standar value with score ${unitTestGetValue}/${unitTest_standard}"
                // } else {
                //     currentBuild.result = 'ABORTED'
                //     error("Sorry your unit test score not fulfill standard score ${unitTestGetValue}/${unitTest_standard}")
                // }
            }
            
    stage ("SonarQube Analysis") {
        //Set Credentials
        withSonarQubeEnv(credentialsId: 'sonarqube-token', installationName: 'sonarqube') {
            // SonarScanner
            sh """
                sonar-scanner -Dsonar.projectKey=${appName} \
                            -Dsonar.exclusions="**/main.go,**/*.pb.go" \
                            -Dsonar.language=go \
                            -Dsonar.go.coverage.reportPaths=${sonarCoverage} \
                            -Dsonar.test.inclusions="**/*_test.go" \
                            -Dsonar.test.exclusions="**/*.pb.go" \
            """
        }
        //quality gate
        sh "sleep 30"
        timeout(time: 10, unit: 'MINUTES') {
              def qg = waitForQualityGate()
              if (qg.status != 'OK') {
                  error "Pipeline aborted due to quality gate failure: ${qg.status}"
              }
        }
    }

            // ::NONPROD-PIPELINE
            if (version == 'alpha' || version == 'beta') {
                if (version == 'alpha') {
                    echo "Yes, it's dev branch. Continue to DEV Pipeline"
                } /*else if (version == 'beta'){
                stage ("Merge Pull Request") {
                    // GET PULL REQUEST ID
                    sh "curl -H \"PRIVATE-TOKEN: ${approval_token}\" \"https://gitlab.com/api/v4/projects/${projectID}/merge_requests\" --output resultMerge.json"
                    def jsonMerge = readJSON file: "resultMerge.json"                    
                    echo "Request from: ${jsonMerge[0].author.name}"

                    // STATUS VALIDATION
                    if (jsonMerge[0].state == "opened") {
                        // GET ALL COMMENTS ON PULL REQUEST
                        sh "curl -H \"PRIVATE-TOKEN: ${approval_token}\" \"https://gitlab.com/api/v4/projects/${projectID}/merge_requests/${jsonMerge[0].iid}/notes\" --output comment.json"
                        def commentJson = readJSON file: "comment.json"
                        def checking = false

                        // LOOP ALL COMMENT TO GET APPROVAL
                        commentJson.each { res ->

                            // CHECK IF CURRENT INDEX HAS SYSTEM FALSE 
                            if (!res.system && !checking) {
                                // IF COMMENT HAS VALUE: APPROVED AND AUTHOR IS VALID
                                if (res.body == "Approved" && approval.contains(res.author.username)) {
                                    addGitLabMRComment(comment: "Pull Request Approved by Jenkins")
                                    acceptGitLabMR(useMRDescription: true, removeSourceBranch: false)
                                } else {
                                    currentBuild.result = 'ABORTED'
                                    error("Sorry, your approval is not valid")
                                }
                                checking = true
                            }
                        }
                    } else {
                        error("Pull Request ${jsonMerge[0].title} ${jsonMerge[0].iid} is already ${jsonMerge[0].state}. Please Create a new Pull Request")
                    }
                    }
                }*/ else {
                    echo "Sorry, we will not running anyway"
                }

                stage ("Artifact Repository") {
                    parallel (
                        'Container': {
                            stage ("Build Container") {
                            // if (version == 'alpha') {
                                dockerBuild(docker_url: docker_url, image_name: image_name, image_version: version)
                                //dockerBuildGrpc(docker_url: docker_url, image_name_grpc: image_name_grpc, image_version: version)
                            }
                            //else if (version == 'beta') {
                              //  dockerBuildstaging(docker_url: docker_url, image_name: image_name, image_version: version)
                                //dockerBuildGrpcstaging(docker_url: docker_url, image_name_grpc: image_name_grpc, image_version: version)
                               // }
                           // }
                            stage ("Push Container") {
                                docker.withRegistry("https://${docker_url}", docker_credentials) {
                                    //UPDATE TAG
                                    //dockerPushTag(docker_url: docker_url, image_name: image_name, srcVersion: version, dstVersion: "${version}-${BUILD_NUMBER}")
                                    //dockerPushTagGrpc(docker_url: docker_url, image_name_grpc: image_name_grpc, srcVersion: version, dstVersion: "${version}-${BUILD_NUMBER}")
                                    // ::LATEST
                                    //dockerPush(docker_url: docker_url, image_name: image_name, image_version: version)
                                    //dockerPushGrpc(docker_url: docker_url, image_name_grpc: image_name_grpc, image_version: version)
                                    // ::VERSION
                                   // dockerPushTag(docker_url: docker_url, image_name: image_name, srcVersion: version, dstVersion: "${version}-${BUILD_NUMBER}")
                                   // dockerRemove(docker_url: docker_url, image_name: image_name, image_version: "${version}-${BUILD_NUMBER}")
                                    //dockerRemoveGrpc(docker_url: docker_url, image_name_grpc: image_name_grpc, image_version: "${version}-${BUILD_NUMBER}")
                                    if (version == 'alpha') {
                                        dockerPush(docker_url: docker_url, image_name: image_name, image_version: version)
                                        dockerPushTag(docker_url: docker_url, image_name: image_name, srcVersion: version, dstVersion: "${version}-${BUILD_NUMBER}")
                                        dockerRemove(docker_url: docker_url, image_name: image_name, image_version: version)
                                      //  dockerRemoveGrpc(docker_url: docker_url, image_name_grpc: image_name_grpc, image_version: version)
                                    }
                                    // ::REVERT
                                    if (version == 'beta') {
                                        dockerPush(docker_url: docker_url, image_name: image_name, image_version: version)
                                        dockerPushTag(docker_url: docker_url, image_name: image_name, srcVersion: version, dstVersion: "${version}-${BUILD_NUMBER}")
                                        dockerPushTag(docker_url: docker_url, image_name: image_name, srcVersion: version, dstVersion: "${version}-revert")
                                        //dockerPushTagGrpc(docker_url: docker_url, image_name_grpc: image_name_grpc, srcVersion: version, dstVersion: "${version}-revert")
                                        dockerRemove(docker_url: docker_url, image_name: image_name, image_version: "${version}-revert")
                                        //dockerRemoveGrpc(docker_url: docker_url, image_name_grpc: image_name_grpc, image_version: "${version}-revert")
                                        dockerRemove(docker_url: docker_url, image_name: image_name, image_version: version)
                                        //dockerRemoveGrpc(docker_url: docker_url, image_name_grpc: image_name_grpc, image_version: version)
                                    }
                                }
                            }
                        },

                        'Chart': {
                            stage ("Packaging") {
                                dir('helm') {
                                    checkout([$class: 'GitSCM', branches: [[name: '*/master']], userRemoteConfigs: [[credentialsId: 'ci-cd', url: helm_git]]])
                                    dir(helm_dir) {
                                        helmLint(service_name)
                                        //helmLintGrpc(service_name_grpc)
                                        //helmDryRun(name: service_name, service_name: service_name)
                                        //helmDryRunGrpc(name: service_name_grpc, service_name_grpc: service_name_grpc)
                                        helmPackage(service_name)
                                        //helmPackageGrpc(service_name_grpc)
                                    }
                                }
                            }

                            stage ("Push Chart") {
                                echo "Push Chart to Artifactory"
                                 dir('helm') {
                                   dir(helm_dir) {
                                    pushArtifact(name: "helm", service_name: service_name, artifact_url: artifact_url)
                                    //pushArtifactGrpc(name: "helm", service_name_grpc: service_name_grpc, artifact_url: artifact_url)
                                    }
                                }
                            }
                        },
                    )
                }
            } else {
                echo "Sorry we can't continue. Because it's not DEV or Staging cluster"
            }
        } else {
            // ::PRODUCTION
            stage ("Update Container") {
                sh "docker pull ${docker_staging}/${image_name}:beta"
                docker.withRegistry("https://${docker_url}", docker_credentials) {
                    sh "docker tag ${docker_staging}/${image_name}:beta ${docker_prod}/${image_name}:release"
                    dockerPush(docker_url: docker_url, image_name: image_name, image_version: version)
                    dockerPushTag(docker_url: docker_url, image_name: image_name, srcVersion: version, dstVersion: "${version}-revert")
                    dockerRemove(docker_url: docker_url, image_name: image_name, image_version: "${version}-revert")
                    dockerRemove(docker_url: docker_url, image_name: image_name, image_version: version)
                    dockerRemove(docker_url: "${docker_staging}", image_name: image_name, image_version: "beta")
                }
            }

            stage ("Packaging") {
                dir('helm') {
                    checkout([$class: 'GitSCM', branches: [[name: '*/master']], userRemoteConfigs: [[credentialsId: 'ci-cd', url: helm_git]]])
                    dir(helm_dir) {
                        helmLint(service_name)
                        //helmLintGrpc(service_name_grpc)
                        //helmDryRun(name: service_name, service_name: service_name)
                       // helmDryRunGrpc(name: service_name_grpc, service_name_grpc: service_name_grpc)
                        helmPackage(service_name)
                        //helmPackageGrpc(service_name_grpc)
                    }
                }
            }

            stage ("Push Chart") {
                echo "Push Chart to Artifactory"
                dir('helm') {
                    dir(helm_dir) {
                         pushArtifact(name: "helm", service_name: service_name, artifact_url: artifact_url)
                         //pushArtifactGrpc(name: "helm", service_name_grpc: service_name_grpc, artifact_url: artifact_url)
                    }
               }
            }
        }

       // ::VERIFY-DEPLOYMENT
        if (version != "debug") {
            stage ("Deployment") {
              	//deploySpinnaker(spinnaker_webhook: spinnaker_webhook, version: version)
                dir('helm') {
                    dir(helm_dir) {
                        withKubeConfig(credentialsId: credentialsCluster, serverUrl: serverUrl) {
                            try {
                                helmInstall(name: service_name, service_name: service_name)
                            } catch (e) {
                                helmUpgrade(name: service_name, service_name: service_name)
                            }
                       }
                    }
                }
            }
        } else {
            echo "Sorry, No Deployment"
        }

        // ::TESTING
        try {
            if (version == "beta") {
                stage ("Security Test") {
                    try {
                        burpsuiteScan(name_space: "${name_space}", burpsuite_url: "${burpsuite_url}", endpoint_urls: "${endpoint_urls}")
                    } catch (e) {
                        echo "${e}"
                    }
                }


                stage ("Performance Test") {
                    try {
                        echo "Running Performance Test"
                        //node ('k6') {
                        //    dir ('performance') {
                        //      checkout([$class: 'GitSCM', branches: [[name: '*/master']], userRemoteConfigs: [[credentialsId: 'ci-cd', url: k6_repo]]])
                        //      sh "k6 run ${service_name}.js"
                        //    }
                        //}
                    } catch (e) {
                        echo "${e}"
                    }
                }

                stage ("Regression Test") {
                    try {
                        echo 'Running Regression Test'
                        node ('kre-centos') {
                            sleep 60
                            cleanWs deleteDirs: true
                            checkout([$class: 'GitSCM', userRemoteConfigs: [[credentialsId: 'ci-cd', url: "${katalonRepoUrl}"]], branches: [[name: "master"]]])
                            echo "workspace : ${workspace}"
                            regressionTest(workspace: "${workspace}", katalonProjectName: "${katalonProjectName}", katalonTestSuiteName: "${katalonTestSuiteName}")
                        }
                    } catch (e) {
                        echo "${e}"
                    }
                }
            } else if (version == "release") {
                try {
                    stage("Sanity Test") {
                        try {
                            echo "Running Sanity Test"
                        } catch (e) {
                            echo "${e}"
                        }
                    }
                } catch (e) {
                    stage ("Revert Production") {
                        docker.withRegistry("https://${docker_url}", docker_credentials) {
                            dockerPull(docker_url: docker_url, image_name: image_name, image_version: "${version}-revert")
                            dockerPushTag(docker_url: docker_url, image_name: image_name, srcVersion: "${version}-revert", dstVersion: version)
                            dockerRemove(docker_url: docker_url, image_name: image_name, image_version: version)
                        }
                    }

                    stage ("Deployment") {
						//deploySpinnaker(spinnaker_webhook: spinnaker_webhook, version: version)
                        dir('helm') {
                            dir(helm_dir) {
                                withKubeConfig(credentialsId: credentialsCluster, serverUrl: serverUrl) {
                                    helmRollBack(name: service_name, service_name: service_name)
                                }
                            }
                        }
                    }
                }
            } else {
                echo "No test except staging or production"
            }

        } catch (e) {
            // ::REVERT STAGING
            stage ("Revert Previous Merge") {
                echo "Get Merge ID & Push Again"
                // CODE HERE
            }

            stage ("Revert Container") {
                docker.withRegistry("https://${docker_url}", docker_credentials) {
                    dockerPull(docker_url: docker_url, image_name: image_name, image_version: "${version}-revert")
                    dockerPushTag(docker_url: docker_url, image_name: image_name, srcVersion: "${version}-revert", dstVersion: version)
                    dockerRemove(docker_url: docker_url, image_name: image_name, image_version: version)
                }
            }

            stage ("Deployment") {
				//deploySpinnaker(spinnaker_webhook: spinnaker_webhook, version: version)
                dir('helm') {
                    dir(helm_dir) {
                        withKubeConfig(credentialsId: credentialsCluster, serverUrl: serverUrl) {
                            helmRollBack(name: service_name, service_name: service_name)
                        }
                    }
                }
            }
        }

            stage ("Notifications") {
				deleteDir()
                echo "Job Success"
                notifications(telegram_url: telegram_url, telegram_chatid: telegram_chatid, 
                job: env.JOB_NAME, job_numb: env.BUILD_NUMBER, job_url: env.BUILD_URL, job_status: job_success, unitTest_score: unitTest_score
                )
            }
        } catch (e) {

        stage ("Error") {
			deleteDir()
            echo "Job Failed"
            notifications(telegram_url: telegram_url, telegram_chatid: telegram_chatid, 
            job: env.JOB_NAME, job_numb: env.BUILD_NUMBER, job_url: env.BUILD_URL, job_status: job_error, unitTest_score: unitTest_score
            )
        }
    }
}

def sonarScanGo(Map args) {
    sh "sonar-scanner -X \
    -Dsonar.projectName=${args.project_name} \
    -Dsonar.projectKey=${args.image_name} \
    -Dsonar.projectVersion=${args.version} \
    -Dsonar.qualitygate.wait=true \
    -Dsonar.language=go \
    -Dsonar.dynamicAnalysis=reuseReports \
    -Dsonar.exclusions=**/main.go \
    -Dsonar.go.coverage.reportPaths=${args.sonarCoverage} \
    -Dsonar.test.inclusions=**/*_test.go \
    -Dsonar.test.exclusions=**/**.pb.go"
}

def goEnv() {
    sh (
        script: '''
        go version

        rm -rf report.xml
        rm -rf coverage.out
        rm -rf cover
        
        export MIGRATION_PATH=/migrations/test
        export PATH=$PATH:$(go env GOPATH)/bin

        go mod tidy -v

        go get -u golang.org/x/lint/golint
        
        go get -u github.com/jstemmer/go-junit-report
        go clean -testcache
        '''
    )
}

def regressionTest(Map args) {
    withCredentials([string(credentialsId: 'katalon-api-key', variable: 'secret')]) {
        sh "/katalon01/katalon-studio-engine/katalonc -noSplash -runMode=console \
        -projectPath='${args.workspace}/${args.katalonProjectName}' -retry=0 \
        -testSuitePath='Test Suites/${args.katalonTestSuiteName}' -executionProfile=default \
        -browserType='Chrome (headless)' -apiKey='${secret}'"
    }
}

def dockerBuild(Map args) {
    sh "make gen"
    sh "docker build --network host -t ${args.docker_url}/${args.image_name}:${args.image_version} ."
   // sh "make build-module-builder"
   // sh "make create-api"
}

def dockerBuildstaging(Map args) {
    sh "make gen"
    sh "make build-module-builder-staging"
    sh "make create-api-staging"
}

def dockerBuildGrpcstaging(Map args) {
    sh "make gen"
    sh "make build-module-builder-staging"
    sh "make create-grpc-staging"
}

def dockerBuildGrpc(Map args) {
    sh "make create-grpc"
}

def dockerPushTag(Map args) {
    sh "docker tag ${args.docker_url}/${args.image_name}:${args.srcVersion} ${args.docker_url}/${args.image_name}:${args.dstVersion}"
    sh "docker push ${args.docker_url}/${args.image_name}:${args.dstVersion}"
}

//def dockerPushTagGrpc(Map args) {
//    sh "docker tag ${args.docker_url}/${args.image_name_grpc}:alpha ${args.docker_url}/${args.image_name_grpc}:${args.dstVersion}"
//    sh "docker push ${args.docker_url}/${args.image_name_grpc}:${args.dstVersion}"
//}

def dockerPush(Map args) {
  //  sh "docker tag ${args.docker_url}/${args.image_name}:alpha ${args.docker_url}/${args.image_name}:${args.image_version}"
    sh "docker push ${args.docker_url}/${args.image_name}:${args.image_version}"
}

//def dockerPushGrpc(Map args) {
//    sh "docker tag ${args.docker_url}/${args.image_name_grpc}:alpha ${args.docker_url}/${args.image_name_grpc}:${args.image_version}"
//    sh "docker push ${args.docker_url}/${args.image_name_grpc}:${args.image_version}"
//}

def dockerPull(Map args) {
    sh "docker pull ${args.docker_url}/${args.image_name}:${args.image_version}"
}

//def dockerPullGrpc(Map args) {
//    sh "docker pull ${args.docker_url}/${args.image_name_grpc}:${args.image_version}"
//}

def dockerRemove(Map args) {
    sh "docker rmi -f ${args.docker_url}/${args.image_name}:${args.image_version}"
    sh 'docker image prune -a -f'
}

def dockerRemoveGrpc(Map args) {
    sh "docker rmi -f ${args.docker_url}/${args.image_name_grpc}:${args.image_version}"
    sh "docker rmi -f ${args.docker_url}/${args.image_name_grpc}:${args.dstVersion}"
}

def helmLint(String service_name) {
    echo "Running helm lint"
    sh "helm lint ${service_name}"
}

def helmLintGrpc(String service_name_grpc) {
    echo "Running helm lint"
    sh "helm lint ${service_name_grpc}"
}

def helmDryRun(Map args) {
    echo "Running dry-run deployment"
    sh "helm install --dry-run --debug ${args.name} ${args.service_name} -n dummy"
}

def helmDryRunGrpc(Map args) {
    echo "Running dry-run deployment"
    sh "helm install --dry-run --debug ${args.name} ${args.service_name_grpc} -n dummy"
}

def helmPackage(String service_name) {
    echo "Running Helm Package"
    sh "helm package ${service_name}"
}

def helmPackageGrpc(String service_name_grpc) {
    echo "Running Helm Package"
    sh "helm package ${service_name_grpc}"
}

def helmUpgrade(Map args) {
    echo "Upgrade chart deployment"
    sh "helm upgrade --install ${args.name} ${args.service_name} -n dummy"
    sh "kubectl rollout restart deployment ${args.service_name} -n dummy"
}

def helmUpgradeGrpc(Map args) {
    echo "Upgrade chart deployment"
    sh "helm upgrade ${args.name} ${args.service_name_grpc} -n dummy --recreate-pods"
}

def helmInstall(Map args) {
    echo "Install chart deployment"
    sh "helm install ${args.name} ${args.service_name} -n dummy"
}

def helmInstallGrpc(Map args) {
    echo "Install chart deployment"
    sh "helm install ${args.name} ${args.service_name_grpc} -n dummy"
}

def helmRollBack(Map args) {
    echo "Roolback chart deployment to Previous Version"
    sh "helm rollback ${args.name} 0 ${args.service_name} -n dummy"
}

def helmRollBackGrpc(Map args) {
    echo "Roolback chart deployment to Previous Version"
    sh "helm rollback ${args.name} 0 ${args.service_name_grpc} -n dummy"
}

def pushArtifact(Map args) {
    echo "Upload to Artifactory Server"
    if (args.name == "helm") {
        withCredentials([usernamePassword(credentialsId: 'ci-cd', passwordVariable: 'password', usernameVariable: 'username')]) {
            sh "curl -v -n -u ${username}:${password} -T ${args.service_name}-*.tgz ${args.artifact_url}/"   
        }     
    } else {
        withCredentials([usernamePassword(credentialsId: 'ci-cd', passwordVariable: 'password', usernameVariable: 'username')]) {
            sh "curl -v -n -u ${username}:${password} -T ${args.name} ${args.artifact_url}/"
        }
    }
}

def pushArtifactGrpc(Map args) {
    echo "Upload to Artifactory Server"
    if (args.name == "helm") {
        withCredentials([usernamePassword(credentialsId: 'ci-cd', passwordVariable: 'password', usernameVariable: 'username')]) {
            sh "curl -v -n -u ${username}:${password} -T ${args.service_name_grpc}-*.tgz ${args.artifact_url}/"   
        }     
    } else {
        withCredentials([usernamePassword(credentialsId: 'ci-cd', passwordVariable: 'password', usernameVariable: 'username')]) {
            sh "curl -v -n -u ${username}:${password} -T ${args.name} ${args.artifact_url}/"
        }
    }
}

def notifications(Map args) {
    def message = " Dear Team PRH \n CICD Pipeline ${args.job} ${args.job_status} with build ${args.job_numb} \n\n More info at: ${args.job_url} \n\n Unit Test: ${args.unitTest_score} \n\n Total Time : ${currentBuild.durationString}"
    sh "curl -s -X POST ${args.telegram_url} -d chat_id=${args.telegram_chatid} -d text='${message}'"
    //parallel(
    //     "Telegram": {
    //       sh "curl -s -X POST ${args.telegram_url} -d chat_id=${args.telegram_chatid} -d text='${message}'"
    //    },
    //    "Jira": {
            //jiraSend color: "${args.jira_url}", message: "${message}", channel: "${args.slack_channel}"
    //    }
    //)
}

def deploySpinnaker(Map args) {
    def hook = registerWebhook()
    echo "Hi Spinnaker!"
    sh "curl ${args.spinnaker_webhook}-${args.version} -X POST -H 'content-type: application/json' -d '{ \"parameters\": { \"jenkins-url\": \"${hook.getURL()}\" }}'"
    def data = waitForWebhook hook
    echo "Webhook called with data: ${data}"
}
