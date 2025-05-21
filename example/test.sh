#!/bin/bash
echo test{abcd=\"${TEST_ENV}\"} 20
echo test 20
echo test{} 20
echo test_abcd{abcd=\"${TEST_ENV}\"} 20
echo test_abcd\{abcd=\"${TEST_ENV}\",abidsa=\"${TEST_ENV2}\"\} 20
