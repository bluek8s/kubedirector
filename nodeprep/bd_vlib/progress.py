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

from .errors import DescTooLongException, PercentageOutOfRangeException
from .utils import notify_progress

MAX_DESC_STR_LEN=64

def BDVLIB_Progress(percentage, description):
    """
    Reports progress of the configuration. This information is propagated all
    the way to the user's interface.

    Parameters:
        percentage: is a number between 0 and 100 (both inclusive)
        description: a short descriptive string less than 64chars.

    Returns:
        0 on success.
        1 on any failure.
    Exceptions:
        DescTooLongException: The description string provided is too long.
        PercentageOutOfRangeException:
    """
    if len(description) > MAX_DESC_STR_LEN:
        raise DescTooLongException()

    if (int(percentage) > 100) or (int(percentage) < 0):
        raise PercentageOutOfRangeException()

    return notify_progress(percentage, description)
