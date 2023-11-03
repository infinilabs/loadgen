# runner: {
#   // total_rounds: 1
#   no_warm: false,
#   // Whether to log all requests
#   log_requests: false,
#   // Whether to log all requests with the specified response status
#   log_status_codes: [0, 500],
#   assert_invalid: false,
#   assert_error: false,
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

POST $[[env.ES_ENDPOINT]]/medcl/_search
{ "track_total_hits": true, "size": 0, "query": { "terms": { "patent_id": [ $[[id_list]] ] } } }
# request: {
#   runtime_variables: {batch_no: "uuid"},
#   runtime_body_line_variables: {routing_no: "uuid"},
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
