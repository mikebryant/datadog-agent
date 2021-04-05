#
# Copyright 2012-2014 Chef Software, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

require './lib/cmake.rb'

name "zstd"
default_version "1.4.5"

license "BSD"
license_file "LICENSE"
skip_transitive_dependency_licensing true

dependency "libarchive"

version("1.4.5") { source sha256: "98e91c7c6bf162bf90e4e70fdbc41a8188b9fa8de5ad840c401198014406ce9e" }

source url: "https://github.com/facebook/zstd/releases/download/v#{version}/zstd-#{version}.tar.gz"

relative_path "zstd-#{version}"

build do
  env = with_standard_compiler_flags(with_embedded_path)

  cmake_build_dir = "#{project_dir}/build/cmake/builddir"

  command "mkdir #{cmake_build_dir}", env: env

  cmake_options = [
    "-DZSTD_BUILD_PROGRAMS=OFF"
  ]

  cmake(*cmake_options, env: env, cwd: cmake_build_dir, prefix: "#{install_dir}/embedded")
end
