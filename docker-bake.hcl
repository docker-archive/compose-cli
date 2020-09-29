variable "GIT_TAG" {
    default = "dev"
}

variable "BUILD_TAGS" {
    default = ""
}

function "validate-targets" {
    params = []
    result = ["validate-headers", "validate-go-mod"]
}

function "dev-build-args" {
    params = [tag]
    result = {
        BUILD_TAGS = "example,local,ecs"
        GIT_TAG = "${tag}"
    }
}

group "default" {
	targets = ["cli"]
}

group "validate" {
    targets = validate-targets()
}

group "check" {
    targets = concat(validate-targets(), ["import-restrictions", "test", "lint"])
}

target "cli" {
    output = ["./bin"]
    target = "cli"
    platforms = ["local"]
    args = dev-build-args("${GIT_TAG}")
}

target "cross" {
    output = ["./bin"]
    target = "cross"
    args = {
        BUILD_TAGS = "${BUILD_TAGS}"
        GIT_TAG = "${GIT_TAG}"
    }
}

target "test" {
    target = "test"
    args = dev-build-args("${GIT_TAG}")
}

target "lint" {
    target = "lint"
    args = {
        GIT_TAG = "${GIT_TAG}"
    }
}

target "import-restrictions" {
    target = "import-restrictions"
}

target "validate-headers" {
    target = "check-license-headers"
}

target "validate-go-mod" {
    target = "check-go-mod"
}

target "protos" {
    output = ["./protos"]
    target = "protos"
}
