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

name "glib"
default_version "2.66.1"

license "LGPL-2.1"
license_file "COPYING"
skip_transitive_dependency_licensing true

dependency "libffi"
dependency "pcre"
dependency "elfutils"

version("2.66.1") { source sha256: "5ee680d0b943e47dc1bd53391611a68dca811e2ad635fe0c397db6f250006984" }

source url: "https://gitlab.gnome.org/GNOME/glib/-/archive/#{version}/glib-#{version}.tar.bz2"

relative_path "glib-#{version}"

build do
  env = with_standard_compiler_flags(with_embedded_path)

  patch source: "0001-Set-dependency-method-to-pkg-config.patch", env: env

  meson_command = [
    "meson",
    "_build",
    "--prefix=#{install_dir}/embedded",
    "--libdir=lib",
    "-Dlibmount=disabled",
    "-Dselinux=disabled",
  ]

  command meson_command.join(" "), env: env

  command "ninja -C _build", env: env
  command "ninja -C _build install", env: env
end
