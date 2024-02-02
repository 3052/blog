# Electric provider

Check your bill, find the Energy Charge, example:

> $0.125 per kWh

and the average price, example:

> $0.173 per kWh

then click the Fact Sheet for another company. Find the Energy Charge, example:

> $0.07752

and the average price, example:

> $0.125

1. <http://powertochoose.org>
2. Click export results to Excel

~~~
xlsx2csv -N power-to-choose.xlsx
xlsx2csv -o power.csv power-to-choose.xlsx Content
~~~

- <https://github.com/caltechlibrary/datatools>
- <https://sqlite.org/cli.html#importing_csv_files>

~~~
0.128|0.124|True|1|1.532|FRONTIER UTILITIES
$225

1.568|True|0.127|0.131|2|GEXA ENERGY
$225

0.128|0.125|True|-1|1.533|OhmConnect Energy
$400

0.128|0.124|False|2|1.532|RHYTHM
$150

0.128|0.125|True|2|1.533|Value Power
$225

...>    (11 * "Price/kWh 500" + "Price/kWh 1000") as price,
...>    "New Customer",
...>    "Price/kWh 500",
...>    "Price/kWh 1000",
...>    "Rating",
...>    RepCompany
1.569|True|0.131|0.128|3|AMIGO ENERGY
1.569|True|0.131|0.128|2|JUST ENERGY
1.569|True|0.131|0.128|3|TARA ENERGY
1.581|True|0.132|0.129|3|AMIGO ENERGY
1.581|True|0.132|0.129|2|JUST ENERGY
1.581|True|0.132|0.129|3|TARA ENERGY
1.587|False|0.133|0.124|-1|GOOD CHARLIE & CO LLC
1.588|False|0.133|0.125|2|SOUTHERN FEDERAL POWER LLC
1.609|True|0.135|0.124|1|TRUE POWER
1.616|False|0.135|0.131|1|PowerNext
1.628|False|0.136|0.132|4|Shell Energy Solutions
1.651|False|0.138|0.133|1|PowerNext
1.653|False|0.138|0.135|3|CONSTELLATION NEWENERGY INC
1.66|False|0.139|0.131|3|Energy Texas
1.691|True|0.142|0.129|1|TRUE POWER
1.7|True|0.142|0.138|1|CleanSky Energy
1.701|True|0.142|0.139|3|AMIGO ENERGY
1.701|True|0.142|0.139|2|JUST ENERGY
1.701|True|0.142|0.139|3|TARA ENERGY
1.707|False|0.143|0.134|-1|GOOD CHARLIE & CO LLC
1.713|True|0.143|0.14|3|AMIGO ENERGY
1.713|True|0.143|0.14|2|JUST ENERGY
1.713|True|0.143|0.14|3|TARA ENERGY
1.731|False|0.145|0.136|-1|GOOD CHARLIE & CO LLC
1.737|False|0.145|0.142|2|SOUTHERN FEDERAL POWER LLC
1.755|False|0.147|0.138|-1|GOOD CHARLIE & CO LLC
1.773|True|0.148|0.145|3|AMIGO ENERGY
1.773|True|0.148|0.145|3|TARA ENERGY
1.773|True|0.148|0.145|2|JUST ENERGY
1.784|True|0.149|0.145|1|Veteran Energy
1.796|False|0.15|0.146|5|CHAMPION ENERGY SERVICES LLC
1.796|False|0.15|0.146|1|PowerNext
1.799|True|0.151|0.138|1|TRUE POWER
1.808|False|0.151|0.147|4|Shell Energy Solutions
1.809|True|0.151|0.148|1|CleanSky Energy
1.811|True|0.152|0.139|1|TRUE POWER
1.828|False|0.153|0.145|3|Energy Texas
1.831|False|0.153|0.148|1|PowerNext
1.843|False|0.154|0.149|5|VARSITY ENERGY LLC
1.844|False|0.154|0.15|-1|Flagship Power
1.857|False|0.155|0.152|2|SOUTHERN FEDERAL POWER LLC
1.868|True|0.156|0.152|3|SPARK ENERGY LLC
1.88|False|0.157|0.153|4|Shell Energy Solutions
1.891|False|0.158|0.153|1|PULSE POWER LLC
1.892|False|0.158|0.154|5|CHAMPION ENERGY SERVICES LLC
1.899|False|0.159|0.15|-1|GOOD CHARLIE & CO LLC
1.904|False|0.159|0.155|2|PAYLESS POWER
1.904|False|0.159|0.155|-1|Flagship Power
1.904|False|0.159|0.155|5|BRANCH ENERGY (TEXAS) LLC
1.912|False|0.16|0.152|3|Energy Texas
1.918|True|0.161|0.147|1|TRUE POWER
1.928|True|0.161|0.157|1|Veteran Energy
1.928|True|0.161|0.157|1|Veteran Energy
1.951|False|0.163|0.158|1|PULSE POWER LLC
1.952|True|0.163|0.159|1|4CHANGE ENERGY
1.964|False|0.164|0.16|5|CHAMPION ENERGY SERVICES LLC
1.972|False|0.165|0.157|3|Energy Texas
1.972|True|0.165|0.157|2|Ranchero Power
1.986|False|0.166|0.16|4|YEP
1.997|False|0.167|0.16|4|SOUTHWEST POWER & LIGHT
2.012|True|0.168|0.164|5|THINK ENERGY
2.021|False|0.169|0.162|4|SOUTHWEST POWER & LIGHT
2.022|False|0.169|0.163|4|YEP
2.025|False|0.169|0.166|-1|Flagship Power
2.044|False|0.171|0.163|2|SOUTHERN FEDERAL POWER LLC
2.044|False|0.171|0.163|2|Ranchero Power
~~~
