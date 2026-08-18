# This file is only used as part of 'go test'.

cmake_minimum_required(VERSION 3.10)
project(editorconfig-core-test NONE)

enable_testing()
set(EDITORCONFIG_CMD $ENV{EDITORCONFIG_CMD})
add_subdirectory(core-test)
