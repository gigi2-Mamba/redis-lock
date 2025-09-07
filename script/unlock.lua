# unlock 怎么写呢，常用的lua写法
# lua invoke redis就需要redis.call

# success will return ok, otherwise 0
if redis.call("get",KEYS[1]) == ARGV[1]
then
    return  redis.call("del",KEYS[1])
else
    return 0
end