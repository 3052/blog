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
