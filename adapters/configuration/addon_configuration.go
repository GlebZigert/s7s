package configuration

func (cfg *Configuration) GetUser_for_Axxon(id int64) *User {
    user, _ := cfg.GetUser(id)
    return user
}
