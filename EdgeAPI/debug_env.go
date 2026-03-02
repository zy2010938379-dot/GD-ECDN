package main

import (
	"fmt"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/dbs"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	fmt.Printf("Initial Tea.Env: %s\n", Tea.Env)
	
	// 模拟setup命令的环境设置
	Tea.Env = "dev"
	fmt.Printf("After setting Tea.Env to 'dev': %s\n", Tea.Env)
	
	// 尝试加载数据库配置
	config := &dbs.Config{}
	
	for _, filename := range []string{".db.yaml", "db.yaml"} {
		configPath := Tea.ConfigFile(filename)
		fmt.Printf("Trying to load config from: %s\n", configPath)
		
		configData, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}
		
		err = yaml.Unmarshal(configData, config)
		if err != nil {
			fmt.Printf("Error parsing YAML: %v\n", err)
			continue
		}
		
		fmt.Printf("Successfully loaded config. DBs keys: ")
		for env := range config.DBs {
			fmt.Printf("%s ", env)
		}
		fmt.Println()
		
		// 检查当前环境是否存在
		if dbConfig, ok := config.DBs[Tea.Env]; ok {
			fmt.Printf("Found config for environment '%s': %+v\n", Tea.Env, dbConfig)
		} else {
			fmt.Printf("Config for environment '%s' not found. Available environments: ", Tea.Env)
			for env := range config.DBs {
				fmt.Printf("%s ", env)
			}
			fmt.Println()
		}
		return
	}
	
	fmt.Println("Could not load any database config file")
}