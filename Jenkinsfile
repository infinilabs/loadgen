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
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make clean config build-linux build-arm'
                    sh label: 'package-linux64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux64.tar.gz loadgen-linux64 loadgen.yml'
                    sh label: 'package-linux32', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux32.tar.gz loadgen-linux32 loadgen.yml'
                    sh label: 'package-arm5', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-arm5.tar.gz loadgen-armv5 loadgen.yml'
                    sh label: 'package-arm6', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-arm6.tar.gz loadgen-armv6 loadgen.yml'
                    sh label: 'package-arm7', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-arm7.tar.gz loadgen-armv7 loadgen.yml'
                    sh label: 'package-arm64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-arm64.tar.gz loadgen-arm64 loadgen.yml'
                    archiveArtifacts artifacts: 'loadgen-$VERSION-$BUILD_NUMBER-*.tar.gz', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
                }
            }
        }

    }
}
