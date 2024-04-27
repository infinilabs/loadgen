# // How to use this example?
# // $ INDEX_NAME=medcl123 ES_ENDPOINT=https://localhost:9200 ES_USERNAME=admin  ES_PASSWORD=b14612393da0d4e7a70b ./bin/loadgen -run bulk.dsl

# runner: {
#   total_rounds: 100000,
#   no_warm: true,
#   assert_invalid: true,
#   continue_on_assert_invalid: true,
# }


POST /_bulk
{"index": {"_index": "$[[env.INDEX_NAME]]", "_type": "_doc", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "routing": "$[[routing_no]]", "batch": "$[[batch_no]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
# request: {
#   runtime_variables: {batch_no: "uuid"},
#   runtime_body_line_variables: {routing_no: "uuid"},
#   body_repeat_times: 1000,
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },