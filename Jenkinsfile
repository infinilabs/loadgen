pipeline {

    agent none

    environment { 
        CI = 'true'
    }
    stages {
        
        stage('Build Linux Packages') {

            agent {
                label 'linux'
            }

            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make clean config build-linux'
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make clean config build-arm'
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make clean config build-darwin'
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make clean config build-win'
                    sh label: 'package-linux64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux64.tar.gz loadgen-linux64 loadgen.yml'
                    sh label: 'package-linux32', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux32.tar.gz loadgen-linux32 loadgen.yml'
                    sh label: 'package-arm5', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-arm5.tar.gz loadgen-armv5 loadgen.yml'
                    sh label: 'package-arm6', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-arm6.tar.gz loadgen-armv6 loadgen.yml'
                    sh label: 'package-arm7', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-arm7.tar.gz loadgen-armv7 loadgen.yml'
                    sh label: 'package-arm64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-arm64.tar.gz loadgen-arm64 loadgen.yml'

                    sh label: 'package-mac-amd64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && zip -r ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-mac-amd64.zip loadgen-mac-amd64 loadgen.yml'
                    sh label: 'package-mac-arm64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && zip -r ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-mac-arm64.zip loadgen-mac-arm64 loadgen.yml'

                    sh label: 'package-win-amd64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && zip -r ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-windows-amd64.zip loadgen-windows-amd64.exe loadgen.yml'
                    sh label: 'package-win-386', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && zip -r ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-windows-386.zip loadgen-windows-386.exe loadgen.yml'


                    fingerprint 'loadgen-$VERSION-$BUILD_NUMBER-*'
                    archiveArtifacts artifacts: 'loadgen-$VERSION-$BUILD_NUMBER-*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
                }
            }
        }

    }
}
