#!/usr/bin/env bash
set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )


testCases=(

   "testbindinggithub.sh"
   "testbindinggitlab.sh"
   "testbindingquay.sh"
   "upload_204.sh"
   "testgetconent_private_github.sh"
   "test_sa_basic_auth.sh"
  )


function runSuite {
  for testCase in "${testCases[@]}"; do
    ${SCRIPT_DIR}'/../'login-toolchain.sh $1  $2
    echo '---testcase '"${testCase}"' start ----- at '$(kubectx --current)
    
    if [ "$TEST_RESTART_SPI" = true ] ; then
      restartSPI
    fi

    TEST_PRINT_INFO=${TEST_PRINT_INFO} TEST_ITERATION_NUMBER=${TEST_ITERATION_NUMBER} bash ${SCRIPT_DIR}'/'${testCase}    
    if [ "$TEST_DUMP_METRICS" = true ] ; then
      dumpMetrics
    fi

    echo '---testcase '${testCase}' end ----- at '$(kubectx --current)
  done
}

#kubectl set env deployment/spi-controller-manager TOKENMETADATACACHETTL=1m -n spi-system
#runSuite "$RH_REFRESH_TOKEN" >> ${SCRIPT_DIR}/generic-sute-$(date '+%Y_%m_%d__%H_%M_%S').txt

REGISTRATION_HOST_PROD='https://registration-service-toolchain-host-operator.apps.stone-prd-host1.wdlc.p1.openshiftapps.com/'
REGISTRATION_HOST_STG='https://registration-service-toolchain-host-operator.apps.stone-stg-host.qc0p.p1.openshiftapps.com/'


runSuite $REGISTRATION_HOST_STG "$RH_SKABASHN_USR_TOKEN" >> ${SCRIPT_DIR}/generic-sute-$(date '+%Y_%m_%d__%H_%M_%S').txt
runSuite $REGISTRATION_HOST_STG "$RH_REFRESH_TOKEN" >> ${SCRIPT_DIR}/generic-sute-$(date '+%Y_%m_%d__%H_%M_%S').txt


runSuite $REGISTRATION_HOST_PROD "$RH_SKABASHN_USR_TOKEN" >> ${SCRIPT_DIR}/generic-sute-$(date '+%Y_%m_%d__%H_%M_%S').txt
runSuite $REGISTRATION_HOST_PROD "$RH_REFRESH_TOKEN" >> ${SCRIPT_DIR}/generic-sute-$(date '+%Y_%m_%d__%H_%M_%S').txt

