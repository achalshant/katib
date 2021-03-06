# Generated by the gRPC Python protocol compiler plugin. DO NOT EDIT!
import grpc

import dbif_pb2 as dbif__pb2


class DBIFStub(object):
  """*
  Service for Main API for Katib
  For each RPC service, we define mapping to HTTP REST API method.
  The mapping includes the URL path, query parameters and request body.
  https://cloud.google.com/service-infrastructure/docs/service-management/reference/rpc/google.api#http
  """

  def __init__(self, channel):
    """Constructor.

    Args:
      channel: A grpc.Channel.
    """
    self.RegisterExperiment = channel.unary_unary(
        '/api.v1.alpha2.DBIF/RegisterExperiment',
        request_serializer=dbif__pb2.RegisterExperimentRequest.SerializeToString,
        response_deserializer=dbif__pb2.RegisterExperimentReply.FromString,
        )
    self.DeleteExperiment = channel.unary_unary(
        '/api.v1.alpha2.DBIF/DeleteExperiment',
        request_serializer=dbif__pb2.DeleteExperimentRequest.SerializeToString,
        response_deserializer=dbif__pb2.DeleteExperimentReply.FromString,
        )
    self.GetExperiment = channel.unary_unary(
        '/api.v1.alpha2.DBIF/GetExperiment',
        request_serializer=dbif__pb2.GetExperimentRequest.SerializeToString,
        response_deserializer=dbif__pb2.GetExperimentReply.FromString,
        )
    self.GetExperimentList = channel.unary_unary(
        '/api.v1.alpha2.DBIF/GetExperimentList',
        request_serializer=dbif__pb2.GetExperimentListRequest.SerializeToString,
        response_deserializer=dbif__pb2.GetExperimentListReply.FromString,
        )
    self.UpdateExperimentStatus = channel.unary_unary(
        '/api.v1.alpha2.DBIF/UpdateExperimentStatus',
        request_serializer=dbif__pb2.UpdateExperimentStatusRequest.SerializeToString,
        response_deserializer=dbif__pb2.UpdateExperimentStatusReply.FromString,
        )
    self.UpdateAlgorithmExtraSettings = channel.unary_unary(
        '/api.v1.alpha2.DBIF/UpdateAlgorithmExtraSettings',
        request_serializer=dbif__pb2.UpdateAlgorithmExtraSettingsRequest.SerializeToString,
        response_deserializer=dbif__pb2.UpdateAlgorithmExtraSettingsReply.FromString,
        )
    self.GetAlgorithmExtraSettings = channel.unary_unary(
        '/api.v1.alpha2.DBIF/GetAlgorithmExtraSettings',
        request_serializer=dbif__pb2.GetAlgorithmExtraSettingsRequest.SerializeToString,
        response_deserializer=dbif__pb2.GetAlgorithmExtraSettingsReply.FromString,
        )
    self.RegisterTrial = channel.unary_unary(
        '/api.v1.alpha2.DBIF/RegisterTrial',
        request_serializer=dbif__pb2.RegisterTrialRequest.SerializeToString,
        response_deserializer=dbif__pb2.RegisterTrialReply.FromString,
        )
    self.DeleteTrial = channel.unary_unary(
        '/api.v1.alpha2.DBIF/DeleteTrial',
        request_serializer=dbif__pb2.DeleteTrialRequest.SerializeToString,
        response_deserializer=dbif__pb2.DeleteTrialReply.FromString,
        )
    self.GetTrialList = channel.unary_unary(
        '/api.v1.alpha2.DBIF/GetTrialList',
        request_serializer=dbif__pb2.GetTrialListRequest.SerializeToString,
        response_deserializer=dbif__pb2.GetTrialListReply.FromString,
        )
    self.GetTrial = channel.unary_unary(
        '/api.v1.alpha2.DBIF/GetTrial',
        request_serializer=dbif__pb2.GetTrialRequest.SerializeToString,
        response_deserializer=dbif__pb2.GetTrialReply.FromString,
        )
    self.UpdateTrialStatus = channel.unary_unary(
        '/api.v1.alpha2.DBIF/UpdateTrialStatus',
        request_serializer=dbif__pb2.UpdateTrialStatusRequest.SerializeToString,
        response_deserializer=dbif__pb2.UpdateTrialStatusReply.FromString,
        )
    self.ReportObservationLog = channel.unary_unary(
        '/api.v1.alpha2.DBIF/ReportObservationLog',
        request_serializer=dbif__pb2.ReportObservationLogRequest.SerializeToString,
        response_deserializer=dbif__pb2.ReportObservationLogReply.FromString,
        )
    self.GetObservationLog = channel.unary_unary(
        '/api.v1.alpha2.DBIF/GetObservationLog',
        request_serializer=dbif__pb2.GetObservationLogRequest.SerializeToString,
        response_deserializer=dbif__pb2.GetObservationLogReply.FromString,
        )
    self.SelectOne = channel.unary_unary(
        '/api.v1.alpha2.DBIF/SelectOne',
        request_serializer=dbif__pb2.SelectOneRequest.SerializeToString,
        response_deserializer=dbif__pb2.SelectOneReply.FromString,
        )


class DBIFServicer(object):
  """*
  Service for Main API for Katib
  For each RPC service, we define mapping to HTTP REST API method.
  The mapping includes the URL path, query parameters and request body.
  https://cloud.google.com/service-infrastructure/docs/service-management/reference/rpc/google.api#http
  """

  def RegisterExperiment(self, request, context):
    """*
    Register a Experiment to DB.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def DeleteExperiment(self, request, context):
    """* 
    Delete a Experiment from DB by name.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def GetExperiment(self, request, context):
    """* 
    Get a Experiment from DB by name.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def GetExperimentList(self, request, context):
    """*
    Get a summary list of Experiment from DB.
    The summary includes name and condition.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def UpdateExperimentStatus(self, request, context):
    """* 
    Update Status of a experiment.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def UpdateAlgorithmExtraSettings(self, request, context):
    """* 
    Update AlgorithmExtraSettings.
    The ExtraSetting is created if it does not exist, otherwise it is overwrited.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def GetAlgorithmExtraSettings(self, request, context):
    """* 
    Get all AlgorithmExtraSettings.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def RegisterTrial(self, request, context):
    """*
    Register a Trial to DB.
    ID will be filled by manager automatically.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def DeleteTrial(self, request, context):
    """* 
    Delete a Trial from DB by ID.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def GetTrialList(self, request, context):
    """* 
    Get a list of Trial from DB by name of a Experiment.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def GetTrial(self, request, context):
    """*
    Get a Trial from DB by ID of Trial.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def UpdateTrialStatus(self, request, context):
    """* 
    Update Status of a trial.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def ReportObservationLog(self, request, context):
    """* 
    Report a log of Observations for a Trial.
    The log consists of timestamp and value of metric.
    Katib store every log of metrics.
    You can see accuracy curve or other metric logs on UI.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def GetObservationLog(self, request, context):
    """*
    Get all log of Observations for a Trial.
    """
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def SelectOne(self, request, context):
    # missing associated documentation comment in .proto file
    pass
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')


def add_DBIFServicer_to_server(servicer, server):
  rpc_method_handlers = {
      'RegisterExperiment': grpc.unary_unary_rpc_method_handler(
          servicer.RegisterExperiment,
          request_deserializer=dbif__pb2.RegisterExperimentRequest.FromString,
          response_serializer=dbif__pb2.RegisterExperimentReply.SerializeToString,
      ),
      'DeleteExperiment': grpc.unary_unary_rpc_method_handler(
          servicer.DeleteExperiment,
          request_deserializer=dbif__pb2.DeleteExperimentRequest.FromString,
          response_serializer=dbif__pb2.DeleteExperimentReply.SerializeToString,
      ),
      'GetExperiment': grpc.unary_unary_rpc_method_handler(
          servicer.GetExperiment,
          request_deserializer=dbif__pb2.GetExperimentRequest.FromString,
          response_serializer=dbif__pb2.GetExperimentReply.SerializeToString,
      ),
      'GetExperimentList': grpc.unary_unary_rpc_method_handler(
          servicer.GetExperimentList,
          request_deserializer=dbif__pb2.GetExperimentListRequest.FromString,
          response_serializer=dbif__pb2.GetExperimentListReply.SerializeToString,
      ),
      'UpdateExperimentStatus': grpc.unary_unary_rpc_method_handler(
          servicer.UpdateExperimentStatus,
          request_deserializer=dbif__pb2.UpdateExperimentStatusRequest.FromString,
          response_serializer=dbif__pb2.UpdateExperimentStatusReply.SerializeToString,
      ),
      'UpdateAlgorithmExtraSettings': grpc.unary_unary_rpc_method_handler(
          servicer.UpdateAlgorithmExtraSettings,
          request_deserializer=dbif__pb2.UpdateAlgorithmExtraSettingsRequest.FromString,
          response_serializer=dbif__pb2.UpdateAlgorithmExtraSettingsReply.SerializeToString,
      ),
      'GetAlgorithmExtraSettings': grpc.unary_unary_rpc_method_handler(
          servicer.GetAlgorithmExtraSettings,
          request_deserializer=dbif__pb2.GetAlgorithmExtraSettingsRequest.FromString,
          response_serializer=dbif__pb2.GetAlgorithmExtraSettingsReply.SerializeToString,
      ),
      'RegisterTrial': grpc.unary_unary_rpc_method_handler(
          servicer.RegisterTrial,
          request_deserializer=dbif__pb2.RegisterTrialRequest.FromString,
          response_serializer=dbif__pb2.RegisterTrialReply.SerializeToString,
      ),
      'DeleteTrial': grpc.unary_unary_rpc_method_handler(
          servicer.DeleteTrial,
          request_deserializer=dbif__pb2.DeleteTrialRequest.FromString,
          response_serializer=dbif__pb2.DeleteTrialReply.SerializeToString,
      ),
      'GetTrialList': grpc.unary_unary_rpc_method_handler(
          servicer.GetTrialList,
          request_deserializer=dbif__pb2.GetTrialListRequest.FromString,
          response_serializer=dbif__pb2.GetTrialListReply.SerializeToString,
      ),
      'GetTrial': grpc.unary_unary_rpc_method_handler(
          servicer.GetTrial,
          request_deserializer=dbif__pb2.GetTrialRequest.FromString,
          response_serializer=dbif__pb2.GetTrialReply.SerializeToString,
      ),
      'UpdateTrialStatus': grpc.unary_unary_rpc_method_handler(
          servicer.UpdateTrialStatus,
          request_deserializer=dbif__pb2.UpdateTrialStatusRequest.FromString,
          response_serializer=dbif__pb2.UpdateTrialStatusReply.SerializeToString,
      ),
      'ReportObservationLog': grpc.unary_unary_rpc_method_handler(
          servicer.ReportObservationLog,
          request_deserializer=dbif__pb2.ReportObservationLogRequest.FromString,
          response_serializer=dbif__pb2.ReportObservationLogReply.SerializeToString,
      ),
      'GetObservationLog': grpc.unary_unary_rpc_method_handler(
          servicer.GetObservationLog,
          request_deserializer=dbif__pb2.GetObservationLogRequest.FromString,
          response_serializer=dbif__pb2.GetObservationLogReply.SerializeToString,
      ),
      'SelectOne': grpc.unary_unary_rpc_method_handler(
          servicer.SelectOne,
          request_deserializer=dbif__pb2.SelectOneRequest.FromString,
          response_serializer=dbif__pb2.SelectOneReply.SerializeToString,
      ),
  }
  generic_handler = grpc.method_handlers_generic_handler(
      'api.v1.alpha2.DBIF', rpc_method_handlers)
  server.add_generic_rpc_handlers((generic_handler,))
