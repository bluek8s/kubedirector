#!/usr/bin/env python
#
# Copyright 2018 BlueData Software, Inc.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from distutils.core import setup
from setuptools import find_packages

import configcli

setup(
    name = 'configcli',
    packages = find_packages(),
    version = configcli.__version__,
    description = '',

    zip_safe=False,
    include_package_data=True,

    author = 'BlueData Software, Inc.',
    author_email = 'support@bluedata.com',
    url = 'https://github.com/bluek8s/kubedirector/nodeprep',
    keywords = [ 'configcli', 'configmacro', 'k8s', 'kubedirector'],

    entry_points = {
        "console_scripts" : [
                              'configcli=configcli.__main__:main',
                              'configmacro=configcli.__macro_main__:main'
                            ],
    },
    install_requires = [
    ],
    classifiers = [
            "Environment :: Console",
            "Natural Language :: English",
            "Programming Language :: Python",
            "Intended Audience :: Developers",
            "Development Status :: 5 - Production/Stable",
            "License :: OSI Approved :: Apache Software License",
            "Programming Language :: Python :: Implementation :: CPython",
    ]
)
