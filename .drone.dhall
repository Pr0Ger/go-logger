let JSON = https://prelude.dhall-lang.org/JSON/package.dhall

let Drone = https://dhall.pr0ger.dev/package.dhall

let Enums = https://dhall.pr0ger.dev/enums.dhall

let Misc = https://dhall.pr0ger.dev/misc.dhall

let LintPipeline =
      Drone.Resource.Pipeline.Docker
        Drone.Pipeline.Docker::{
        , name = "lint"
        , steps =
          [ Drone.Step.Docker::{
            , name = "mod tidy"
            , image = "pr0ger/baseimage:build.go-latest"
            , pull = Some Enums.Pull.Always
            , commands =
                Drone.StepType.commands
                  [ "go mod tidy -v", "git diff --exit-code" ]
            }
          , Drone.Step.Docker::{
            , name = "lint"
            , image = "golangci/golangci-lint:v1.39-alpine"
            , commands =
                Drone.StepType.commands
                  [ "go get github.com/golang/mock/mockgen@latest"
                  , "go generate -x"
                  , "golangci-lint run -v"
                  ]
            }
          ]
        }

let TestsPipeline =
      λ(minorVersion : Natural) →
        let minor = Natural/show minorVersion

        in  Drone.Resource.Pipeline.Docker
              Drone.Pipeline.Docker::{
              , name = "tests 1.${minor}"
              , steps =
                [ Drone.Step.Docker::{
                  , name = "build"
                  , image = "pr0ger/baseimage:build.go-1.${minor}"
                  , commands =
                      Drone.StepType.commands
                        [ "go get github.com/golang/mock/mockgen@latest"
                        , "go generate -x"
                        , "go build -v"
                        ]
                  }
                , Drone.Step.Docker::{
                  , name = "test"
                  , image = "pr0ger/baseimage:build.go-1.${minor}"
                  , commands = Drone.StepType.commands [ "go test -v ./..." ]
                  }
                ]
              }

let UpdateDocs =
      Drone.Resource.Pipeline.Docker
        Drone.Pipeline.Docker::{
        , name = "update docs"
        , clone = Some { depth = None Natural, disable = Some True }
        , trigger = Some Misc.Conditions::{
          , event = Some (Misc.ConstraintOrEvent.events [ Enums.Event.Tag ])
          }
        , steps =
          [ Drone.Step.Docker::{
            , name = "pkg.go.dev"
            , image = "alpine:latest"
            , commands =
                Drone.StepType.commands
                  [ "apk add curl jq"
                  , "curl -s https://proxy.golang.org/go.pr0ger.dev/logger/@v/\${DRONE_TAG}.info | jq"
                  ]
            }
          ]
        }

in  Drone.render [ LintPipeline, TestsPipeline 13, UpdateDocs ]
