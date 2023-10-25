# SDS
<img src="https://github.com/stratosnet/sds/blob/main/_stratos-logo-hb-bz.svg" height="100" alt="Stratos Logo"/>  

The Stratos Decentralized Storage (SDS) network is a scalable, reliable, self-balancing elastic acceleration network driven by data traffic. It accesses data efficiently and safely. The user has the full flexibility to store any data regardless of the size and type.

SDS is composed of many resource nodes (also called PP nodes) that store data, and a few metanodes (also called SP nodes) that coordinate everything.
The current repository contains the code for resource nodes only. For more information about metanodes, we will open source it once it's ready.

The latest document on [**how to set up a resource node**](https://docs.thestratos.org/docs-resource-node/setup-and-run-a-sds-resource-node/)
 for our mainnet.

# Contribution

Thank you for considering to help out with the source code! We welcome contributions
from anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to SDS(Stratos Decentralized Storage), please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base.

Please make sure your contributions adhere to our coding guidelines:

* Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting)
  guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
* Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary)
  guidelines.
* Pull requests need to be based on and opened against the `dev` branch, PR name should follow `conventional commits`.
* Commit messages should be prefixed with the package(s) they modify.
    * E.g. "pp: make trace configs optional"

--- ---

# License

Copyright 2023 Stratos

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the [License](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
