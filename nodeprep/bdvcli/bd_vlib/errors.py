#!/bin/env python
#
# Copyright (c) 2015 BlueData Software, Inc.


class KeyTokenListException(Exception):
    """
    The keytoken argument passed is not a list where a list is expected.
    """
    pass

class KeyTokenEmptyException(Exception):
    """
    The keytoken argument passed is an empty list.
    """
    pass

class UnexpectedKeyException(Exception):
    """
    The given key was not expected to resolve to a valid value.
    """
    pass

class KeyLookupException(Exception):
    """
    Generic exception related to a key lookup. The message may provide specific
    error details.
    """
    pass

class KeyTokensRemainingException(Exception):
    """
    Exception raised when the input key token list has more tokens than what
    BD_VLIB can find in its metadata.
    """
    pass

class UnknownValueTypeException(Exception):
    """
    Exception to indicate the value of given key could not be converted to or is
    not a comma separated list of strings.
    """
    pass

class UnknownInputTypeException(Exception):
    """
    The input argument type provided is not recognized by the library.
    """
    pass

class WakeWaitTimeoutException(Exception):
    """
    Timeout expired waiting for a response from another node or set of node(s)
    """
    pass

class DescTooLongException(Exception):
    """
    The description string provided is greater than the accepted length.
    """
    pass

class PercentageOutOfRangeException(Exception):
    """
    The percentage specified is out of range. It expected to be between 0 - 100
    (both inclusive)
    """
    pass

class DesignateTypeUndefined(Exception):
    """
    The designation type specified is unknown.
    """
    pass

class DesignateMissingArgs(Exception):
    """
    A required parameter for a node designation API is missing.
    """
    pass

class UnknownConfigTypeException(Exception):
    """
    The config parameter provided is not an instance of BDVLIB_ConfigMetadata
    """
    pass

class InvalidInputException(Exception):
    """
    Provided input to command in invalid or incomplete.
    """
    pass
