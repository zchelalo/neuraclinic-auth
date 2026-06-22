pipeline {
  agent any

  environment {
    GIT_REPO_URL = 'git@github.com:zchelalo/neuraclinic-auth.git'
    GIT_CREDENTIALS_ID = 'github-ssh-jenkins'
    EMAIL_CREDENTIALS_ID = 'email-credentials'
    SONAR_SCANNER_HOME = tool 'SonarScanner'
  }

  stages {
    stage('Checkout') {
      steps {
        checkout([$class: 'GitSCM',
          branches: [[name: '*/main']],
          userRemoteConfigs: [[
            url: env.GIT_REPO_URL,
            credentialsId: env.GIT_CREDENTIALS_ID
          ]]
        ])
      }
    }

    stage('Go pipeline') {
      agent {
        docker { image 'golang:1.25.5-alpine' }
      }

      stages {
        stage('Install OS Dependencies') {
          steps {
            sh 'apk add --no-cache git ca-certificates'
          }
        }

        stage('Download Dependencies') {
          steps {
            sh 'go mod download'
          }
        }

        stage('Vet') {
          steps {
            sh 'go vet ./...'
          }
        }

        stage('Test') {
          steps {
            sh 'go test ./... -coverprofile=coverage.out'
            stash includes: 'coverage.out', name: 'coverage'
          }
        }

        stage('Build') {
          steps {
            sh 'go build -buildvcs=false -trimpath -o dist/neuraclinic-auth ./cmd'
          }
          post {
            always {
              archiveArtifacts artifacts: 'dist/**', fingerprint: true
            }
          }
        }
      }
    }

    stage('Docker Build') {
      steps {
        sh 'docker build -f .docker/Dockerfile -t neuraclinic-auth:${BUILD_NUMBER} .'
      }
    }

    stage('SonarQube Analysis') {
      steps {
        unstash 'coverage'
        withSonarQubeEnv(credentialsId: 'sonar-token', installationName: 'SonarQube') {
          sh '''
            ${SONAR_SCANNER_HOME}/bin/sonar-scanner \
              -Dsonar.projectKey=neuraclinic-auth \
              -Dsonar.sources=. \
              -Dsonar.exclusions=gen/**,dist/**,certs/**,keys/** \
              -Dsonar.go.coverage.reportPaths=coverage.out
          '''
        }
      }
    }
  }

  post {
    always {
      withCredentials(
        [
          usernamePassword(
            credentialsId: env.EMAIL_CREDENTIALS_ID,
            usernameVariable: 'EMAIL_USER',
            passwordVariable: 'EMAIL_PASS'
          )
        ]
      ) {
        emailext(
          subject: "Build ${currentBuild.currentResult}: ${currentBuild.fullDisplayName}",
          body: "Build ${env.JOB_NAME} (#${env.BUILD_NUMBER}) ${currentBuild.currentResult}.\nMore info: ${env.BUILD_URL}",
          to: env.EMAIL_USER
        )
      }
    }
  }
}
