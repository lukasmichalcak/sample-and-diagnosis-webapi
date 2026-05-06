param (
    $command
)

if (-not $command)  {
    $command = "start"
}

$ProjectRoot = "${PSScriptRoot}/.."

$env:SAMPLE_AND_DIAGNOSIS_API_ENVIRONMENT="Development"
$env:SAMPLE_AND_DIAGNOSIS_API_PORT="8080"
$env:SAMPLE_AND_DIAGNOSIS_API_MONGODB_USERNAME="root"
$env:SAMPLE_AND_DIAGNOSIS_API_MONGODB_PASSWORD="neUhaDnes"

function mongo {
    docker compose --file ${ProjectRoot}/deployments/docker-compose/compose.yaml $args
}

switch ($command) {
    "start" {
        try {
            mongo up --detach
        go run ${ProjectRoot}/cmd/sample-and-diagnosis-api-service
        } finally {
            mongo down
        }
    }
    "mongo" {
        mongo up
    }
    "openapi" {
        docker run --rm -ti -v ${ProjectRoot}:/local openapitools/openapi-generator-cli generate -c /local/scripts/generator-cfg.yaml
    }
    default {
        throw "Unknown command: $command"
    }
}