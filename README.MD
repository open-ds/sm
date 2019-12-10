# SM

SM(String Match)字符串匹配的一个框架。

# Core Data Structure

## Trie tree

数据结构：
```golang
type Node struct {
	IsKey    bool
	Children map[uint8]*Node
	Height   int
	Value    interface{}
	Lock     sync.Mutex
}
```

如果字符集全部是英文，可以用**uint8**，但是如果保存中文字符集，一个中文字符会多创建3个空节点，此时使用**rune**更合适。但是大部分情况是中英文混合的情况。如果分离key的粒度是个问题。

基数树可以解决这个问题，但是实现复杂，插入性能低。

# Replication

节点写加锁的时候是否会影响到读？

## 备份

模仿redis的AOF文件记录对数据的操作记录，AOF文件格式：
```
|*4\r\n|$4|name|\r\n|$6|insert|\r\n|$3|abc|\r\n|$3|abc|\r\n|
|*3\r\n|$4|name|\r\n|$6|remove|\r\n|$3|abc|\r\n|

```

`*4`指该条命令有4个参数，`$4`指参数暂用4个字节。

# Todo List

* [x] 字典树并发插入和删除测试
* [x] 测试HTTP服务的稳定性
* [ ] 实现加载AOF文件的方法
* [ ] 从AOF文件加载数据
* [ ] 实现主从复制
* [ ] 性能检测

