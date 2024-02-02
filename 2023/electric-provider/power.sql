/*
.import --csv power.csv power
*/
select
   (11 * "Price/kWh 500" + "Price/kWh 1000") as price,
   "New Customer",
   "Price/kWh 500",
   "Price/kWh 1000",
   "Rating",
   RepCompany
from power
order by price;
