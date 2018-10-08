#!/usr/bin/env python
#
# Copyright (c) 2018 BlueData Software, Inc.
#
from distutils.core import setup
from setuptools import find_packages

import bdvcli

setup(
    name = 'bdvcli',
    packages = find_packages(),
    version = bdvcli.__version__,
    description = '',

    zip_safe=False,
    include_package_data=True,

    author = 'BlueData Software, Inc.',
    author_email = 'support@bluedata.com',
    url = 'https://github.com/bluedatainc/BlueK8s',
    keywords = [ 'BlueData', 'vcli', 'bdmacro', 'EPIC', 'k8s'],

    entry_points = {
        "console_scripts" : [
                              'bdvcli=bdvcli.__main__:main',
                              'bd_vcli=bdvcli.__main__:main', # For bakward compatibility
                              'bdmacro=bdvcli.__macro_main__:main'
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
