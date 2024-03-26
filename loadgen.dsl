# // How to use DSL to simplify requests, requests defined in loadgen.yml will be skipped in this mode
# // $ES_ENDPOINT=https://localhost:9200 ES_USERNAME=admin  ES_PASSWORD=b14612393da0d4e7a70b ./bin/loadgen -run loadgen.dsl

# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   // Whether to log all requests
#   log_requests: false,
#   // Whether to log all requests with the specified response status
#   log_status_codes: [0, 500],
#   assert_invalid: false,
#   assert_error: false,
#   // Whether to reset the context, including variables, runtime KV pairs,
#   // etc., before this test run.
#   reset_context: false,
#   default_endpoint: "$[[env.ES_ENDPOINT]]",
#   default_basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   }
# },
# variables: [
#   {
#     name: "ip",
#     type: "file",
#     path: "dict/ip.txt",
#     // Replace special characters in the value
#     replace: {
#       '"': '\\"',
#       '\\': '\\\\',
#     },
#   },
#   {
#     name: "id",
#     type: "sequence",
#   },
#   {
#     name: "id64",
#     type: "sequence64",
#   },
#   {
#     name: "uuid",
#     type: "uuid",
#   },
#   {
#     name: "now_local",
#     type: "now_local",
#   },
#   {
#     name: "now_utc",
#     type: "now_utc",
#   },
#   {
#     name: "now_utc_lite",
#     type: "now_utc_lite",
#   },
#   {
#     name: "now_unix",
#     type: "now_unix",
#   },
#   {
#     name: "now_with_format",
#     type: "now_with_format",
#     // https://programming.guide/go/format-parse-string-time-date-example.html
#     format: "2006-01-02T15:04:05-0700",
#   },
#   {
#     name: "suffix",
#     type: "range",
#     from: 10,
#     to: 1000,
#   },
#   {
#     name: "bool",
#     type: "range",
#     from: 0,
#     to: 1,
#   },
#   {
#     name: "list",
#     type: "list",
#     data: ["medcl", "abc", "efg", "xyz"],
#   },
#   {
#     name: "id_list",
#     type: "random_array",
#     variable_type: "number", // number/string
#     variable_key: "suffix", // variable key to get array items
#     square_bracket: false,
#     size: 10, // how many items for array
#   },
#   {
#     name: "str_list",
#     type: "random_array",
#     variable_type: "number", // number/string
#     variable_key: "suffix", // variable key to get array items
#     square_bracket: true,
#     size: 10, // how many items for array
#     replace: {
#       // Use ' instead of " for string quotes
#       '"': "'",
#       // Use {} instead of [] as array brackets
#       "[": "{",
#       "]": "}",
#     },
#   },
# ],

DELETE /medcl

PUT /medcl

POST /medcl/_doc/1
{
 "name": "medcl"
}

POST /medcl/_search
{ "track_total_hits": true, "size": 0, "query": { "terms": { "patent_id": [ $[[id_list]] ] } } }
