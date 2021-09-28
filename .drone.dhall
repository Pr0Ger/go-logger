let JSON = https://prelude.dhall-lang.org/JSON/package.dhall

let List/map = https://prelude.dhall-lang.org/List/map

let List/take = https://prelude.dhall-lang.org/List/take

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

in  Drone.render [ LintPipeline ]
