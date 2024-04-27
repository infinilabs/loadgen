# // How to use this example?
# // $ INDEX_NAME=medcl123 ES_ENDPOINT=https://localhost:9200 ES_USERNAME=admin  ES_PASSWORD=b14612393da0d4e7a70b ./bin/loadgen -run api-testing-example.dsl

# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   assert_invalid: true,
#   continue_on_assert_invalid: true,
# }

DELETE /$[[env.INDEX_NAME]]

PUT /$[[env.INDEX_NAME]]
# 200
# {"acknowledged":true,"shards_acknowledged":true,"index":"medcl123"}

POST /_bulk
{"index": {"_index": "$[[env.INDEX_NAME]]", "_type": "_doc", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "field1": "$[[list]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
{"index": {"_index": "$[[env.INDEX_NAME]]", "_type": "_doc", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "field1": "$[[list]]", "some_other_fields": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
# 200
# {"errors":false,}

GET /$[[env.INDEX_NAME]]/_refresh
# 200
# {"_shards":{"total":2,"successful":1,"failed":0}}

GET /$[[env.INDEX_NAME]]/_count
# 200
# {"count":2}

GET /$[[env.INDEX_NAME]]/_search
# 200
