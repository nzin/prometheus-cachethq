def now = new Date()
env.DATE = now.format("yyyyMMdd_HHmmss")

pipeline {
    agent any
    stages {
        stage('Checkout project...') {
            steps {
                checkout scm
            }
        }

        stage('Define Version') {
            steps {
                script {
                    version = "1.0.0"
                }
            }
        }

        stage('Building...') {
            steps {
                ansiColor('xterm') {
                    sh 'docker build --no-cache .'
                }
            }
        }
        stage('Publishing Docker Image...') {
            when {
                expression {
                    env.BRANCH_NAME == 'master'
                }
            }
            steps {
                dockerBuilder "${env.WORKSPACE}", "prometheus-cachethq", version
            }
        }
    }
}


