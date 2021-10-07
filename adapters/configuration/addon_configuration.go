package configuration

func (cfg *Configuration) GetUser_for_Axxon(id int64) *User {

return cfg.GetUser(id)
}
