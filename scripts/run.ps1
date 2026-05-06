param (
    $command
)

if (-not $command)  {
    $command = "start"
}

$ProjectRoot = "${PSScriptRoot}/.."

$env:SAMPLE_AND_DIAGNOSIS_API_ENVIRONMENT="Development"
$env:SAMPLE_AND_DIAGNOSIS_API_PORT="8080"

switch ($command) {
    "start" {
        go run ${ProjectRoot}/cmd/sample-and-diagnosis-api-service
    }
    "openapi" {
        docker run --rm -ti -v ${ProjectRoot}:/local openapitools/openapi-generator-cli generate -c /local/scripts/generator-cfg.yaml
    }
    default {
        throw "Unknown command: $command"
    }
}