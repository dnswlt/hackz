counter = 0

function request()
  counter = counter + 1
  local id = "item" .. counter
  local body = string.format('{"id":"%s","name":"BenchItem"}', id)
  local headers = {
    ["Content-Type"] = "application/json"
  }
  return wrk.format("POST", "/rpz/items", headers, body)
end
