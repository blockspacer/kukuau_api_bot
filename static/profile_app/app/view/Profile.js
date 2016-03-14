Ext.define('Console.view.Profile', {
    extend: 'Ext.window.Window',
    alias: 'widget.profilewindow',

    title: 'Профайл',
    layout: 'fit',
    autoShow: false,
    autoScroll:true,
    width:800,
    height:900,

    initComponent: function() {
        console.log("init profile");
        var me = this
        this.items = [{
            xtype: 'form',
            items: [ 
            new Ext.form.FormPanel({
                frame: true,
                layout: 'column',
                defaults: {
                    xtype: 'form',
                    columnWidth:0.2,
                    labelAlign: 'top',
                    anchor: '80%'
                },
                items:[
                {
                    xtype: 'image',
                    src:"/img/no_icon.gif",
                    width:100,
                    height:100,
                    name:'image_url',
                    padding:10,
                    id:"profile_image"
                },
                {
                    xtype: 'fileuploadfield',
                    buttonOnly: false,
                    buttonText: "Загрузить",
                    fieldLabel: 'Картинка профиля',
                    name: 'image',
                    padding:10
                },
                ]
            }),
            {
                xtype:'checkbox',
                name:'enabled',
                fieldLabel:"Включен",
                padding:10
            },
            {
                xtype:'checkbox',
                name:'public',
                fieldLabel:"Публичен",
                padding:10
            },
            {
                xtype: 'textfield',
                name : 'name',
                fieldLabel: 'Имя',
                width:400,
                padding:10
            },{
                xtype: 'htmleditor',
                name: 'short_description',
                enableColors:false,
                enableFontSize:false,
                enableLists:false,
                enableSourceEdit:false,
                enableAlignments:false,
                enableFont:false,
                height:70,
                grow: true,
                fieldLabel: 'Слоган',
                padding:10
            },{
                xtype: 'htmleditor',
                name : 'text_description',
                height:100,
                enableFont:false,
                enableColors:false,
                enableFontSize:false,
                enableLists:false,
                enableSourceEdit:false,
                enableAlignments:false,
                grow: true,
                fieldLabel: 'Описание',
                padding:10
            },
            {
                xtype:"grid",
                title:"Контакты",
                itemId:"profile_contacts",
                store: 'ContactsStore',
                columns:[
                {header: 'Тип',  dataIndex: 'type'},
                {header: 'Значение', dataIndex: 'value', flex:1},
                {header: 'Описание', dataIndex: 'description', flex:1},
                {
                    xtype : 'actioncolumn',
                    header : 'Delete',
                    width : 100,
                    align : 'center',
                    action:"delete_contact",
                    items : [
                    {
                        icon:'img/delete-icon.png',
                        tooltip : 'Delete',
                        scope : me
                    }]
                }
                ],
            },
            ]
        }];

        this.dockedItems=[{
            xtype:'toolbar',
            docked: 'top',
            items: [
            {
                text:'Очистить',
                action: 'clear'
            }
            ]
        }];

        this.buttons = [{
            text: 'Сохранить',
            scope: this,
            action: 'save'
        },{
            text:'Удалить',
            action: 'delete',
            scope: this
        },
        {
            text:"Добавить контакт",
            action:"add_contact_start",
            scope: this,
        }

        ];

        this.callParent(arguments);
    }
});