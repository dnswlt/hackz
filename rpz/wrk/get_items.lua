counter = 0
function request()
  counter = counter + 1
  local id = "item" .. (counter % 10000)
  return wrk.format("GET", "/rpz/items/" .. id)
end
