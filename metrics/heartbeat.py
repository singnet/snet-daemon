
from __future__ import print_function
import logging
import threading
import grpc
from grpc_health.v1 import health_pb2 as heartb_pb2
from grpc_health.v1 import health_pb2_grpc as  heartb_pb2_grpc



def run():
    # NOTE(gRPC Python Team): .close() is possible on a channel and should be
    # used in circumstances in which the with statement does not fit the needs
    # of the code.
    response = None
    with grpc.secure_channel('tz-services-1.snet.sh:8008', grpc.ssl_channel_credentials()) as channel:
    #with grpc.secure_channel('asr-ropsten.naint.tech', grpc.ssl_channel_credentials()) as channel:

    #with grpc.insecure_channel('https://example-service-a.singularitynet.io:443') as channel:
        stub = heartb_pb2_grpc.HealthStub(channel)
        try:
            response = stub.Check(heartb_pb2.HealthCheckRequest(service=""))
        except grpc.RpcError as err:
            print(err.code())
            print(err)
    print('==================')
    if response != None:
        print(response.status)
if __name__ == '__main__':
    logging.basicConfig()
    run()
