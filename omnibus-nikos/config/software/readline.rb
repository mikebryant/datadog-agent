#
# Copyright:: Chef Software, Inc.
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

name "readline"
default_version "6.3"

license "GPLv3+"
license_file "COPYING"
skip_transitive_dependency_licensing true

version("6.3") { source sha256: "56ba6071b9462f980c5a72ab0023893b65ba6debb4eeb475d7a563dc65cafd43" }
version("8.0") { source sha256: "e339f51971478d369f8a053a330a190781acb9864cf4c541060f12078948e461" }

source url: "ftp://ftp.gnu.org/gnu/readline/readline-#{version}.tar.gz"

relative_path "readline-#{version}"

build do
  env = with_standard_compiler_flags(with_embedded_path)

  configure_options = [
    "--prefix=#{install_dir}/embedded",
  ]

  configure(*configure_options, env: env)

  make "-j #{workers}", env: env
  make "install", env: env
end
