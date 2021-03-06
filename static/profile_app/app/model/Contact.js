Ext.define("Console.model.Contact",{
	extend:"Ext.data.Model",
	idProperty:'id',
	fields:[
	{name:'id', type:'int'},
	'address',
	'description',
	{name:'lat',  type:'float'},
	{name:'lon',  type:'float'},
	{name:'order_number', type:'int'}
	],
	
	associations: [{
		type: 'hasMany',
		model: 'Console.model.ContactLink',
		name: 'links'
	}],

});