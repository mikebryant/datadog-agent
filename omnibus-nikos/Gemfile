source 'https://rubygems.org'

# Install omnibus
gem "omnibus", git: "https://github.com/ISauve/omnibus", branch: "isabelle.sauve/nikos"

# This development group is installed by default when you run `bundle install`,
# but if you are using Omnibus in a CI-based infrastructure, you do not need
# the Test Kitchen-based build lab. You can skip these unnecessary dependencies
# by running `bundle install --without development` to speed up build times.
group :development do
  # Use Berkshelf for resolving cookbook dependencies
  gem 'berkshelf'

  # Use Test Kitchen with Vagrant for converging the build environment
  gem 'test-kitchen'
  gem 'kitchen-vagrant'
end

