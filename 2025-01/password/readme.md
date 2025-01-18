# password

`password.toml`:

~~~toml
['discord.com']
password = 'alfa bravo'
username = 'alfa.bravo@gmail.com'

['charlie.delta@discord.com']
password = 'charlie delta'
username = 'charlie.delta@protonmail.com'
~~~

usage:

~~~
> password
charlie.delta@discord.com charlie.delta@protonmail.com:charlie delta
discord.com alfa.bravo@gmail.com:alfa bravo

> password discord.com
alfa.bravo@gmail.com:alfa bravo
~~~

## BurntSushi/toml

0 imports

https://github.com/BurntSushi/toml

## freshautomations/stoml

https://github.com/freshautomations/stoml/issues/14

## MinseokOh/toml-cli

https://github.com/MinseokOh/toml-cli/issues/3
