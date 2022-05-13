setup() {
    load '../../../../bats_helpers'

    common_setup
}

@test "spectral" {
    dagger do test
}
