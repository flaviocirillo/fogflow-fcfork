'use strict';

$(function() {

    // initialize the menu bar
    var handlers = {}

    var CurrentScene = null;

    // icon image for device registration
    var iconImage = null;
    var iconImageFileName = null;
    // content image for camera devices
    var contentImage = null;
    var contentImageFileName = null;

    // the list of all registered operators
    var operatorList = [];

    // the list of all available entity type registered by brokers
    var eTypeList = [];

    // design board
    var blocks = null;

    // client to interact with IoT Broker
    var client = new NGSI10Client(config.brokerURL);

    var selectedFogFunction = null;

    addMenuItem('FogFunction', 'Fog Function', showFogFunctions);
    //addMenuItem('TaskInstance', 'Task Instance', showTaskInstances);
   
    initFogFunctionExamples();
    showFogFunctions();
    
    queryOperatorList();
    queryEntityTypeList();


    $(window).on('hashchange', function() {
        var hash = window.location.hash;
        selectMenuItem(location.hash.substring(1));
    });

    function addMenuItem(id, name, func) {
        handlers[id] = func;
        $('#menu').append('<li id="' + id + '"><a href="' + '#' + id + '">' + name + '</a></li>');
    }

    function selectMenuItem(name) {
        $('#menu li').removeClass('active');
        var element = $('#' + name);
        element.addClass('active');

        var handler = handlers[name];        
        if (handler != undefined) {
            handler();
        }
    }

    function initFogFunctionExamples() {
        fetch('/fogfunction').then(res => res.json()).then(fogfunctions => {
            if (Object.keys(fogfunctions).length === 0) {
                fetch("/fogfunction", {
                    method: "POST",
                    headers: {
                        Accept: "application/json",
                        "Content-Type": "application/json"
                    },
                    body: JSON.stringify(myFogFunctionExamples)
                })
                .then(response => {
                    console.log("create the initial set of fog functions: ", response.status)
                    showFogFunctions();
                })
                .catch(err => console.log(err));                                                                
            } else {
                showFogFunctions();
            }               
        })        
    }

    function showFogFunctionEditor(editable) {
        $('#info').html('to design a fog function');

        var html = '';

        html += '<div id="topologySpecification" class="form-horizontal"><fieldset>';

        html += '<div class="control-group"><label class="control-label">name</label>';
        html += '<div class="controls"><input type="text" class="input-large" id="serviceName">';
        html += '</div></div>';

        html += '<div class="control-group"><label class="control-label">description</label>';
        html += '<div class="controls"><textarea class="form-control" rows="3" id="serviceDescription"></textarea>';
        html += '</div></div>';

        if (editable) {
            html += '<div class="control-group"><label class="control-label">topology</label><div class="controls">';
            html += '<span>  </span><button id="cleanBoard" type="button" class="btn btn-default">Clean Board</button>';
            html += '<span>  </span><button id="saveBoard" type="button" class="btn btn-default">Save Board</button>';
            html += '<span>  </span><button id="generateFunction" type="button" class="btn btn-primary">Submit</button>';
            html += '</div></div>';            
        }

        html += '</fieldset></div>';

        html += '<div id="blocks" style="width:800px; height:400px"></div>';

        $('#content').html(html);



        blocks = new Blocks();

        registerAllBlocks(blocks, operatorList, eTypeList);

        blocks.run('#blocks');

        blocks.types.addCompatibility('string', 'choice');

        if (CurrentScene != null) {
            blocks.importData(CurrentScene);
        }

        blocks.ready(function() {
            // associate functions to clickable buttons
            $('#generateFunction').click(function() { //This is Submit button
                boardScene2FogFunction(blocks.export());
            });
            $('#cleanBoard').click(function() {
                blocks.clear();
            });
            $('#saveBoard').click(function() {
                CurrentScene = blocks.export();
            });
        });

        addEditorModifierDecorators(); //This add dynamic actions in the EntityStream popup configurator, for example show Parallelization option when EntityType is selected

    }

    function addEditorModifierDecorators() {// First watch for the modal being added
        const bodyObserver = new MutationObserver(mutations => {
            for (const mutation of mutations) {
                for (const node of mutation.addedNodes) {
                    if (node.nodeType === Node.ELEMENT_NODE && node.classList.contains('blocks_js_modal')) {
                        console.log("Blocks modal detected");
                        // This blocks is the configurator of the entity stream

                        // With this we validate the configurations to be valid
                        validateEntityStreamConfigurator(node)

                        //We this we add dynamicity to the configurator popup
                        handleEntityStreamConfigurator(node);
                    }
                }
            }
        });

        bodyObserver.observe(document.body, { childList: true, subtree: true });

        function validateEntityStreamEntries(configuratorElement) {

            console.log("Validing the EntityStream configuration")

            const groupBy = configuratorElement.querySelector('select[name="groupby"]');
            if (!groupBy) return; // Exit the function if not found because we are not in an EntityStream configuration

            const parallelization = configuratorElement.querySelector('input[name="parallelization"]');
            const numberofinstances = configuratorElement.querySelector('input[name="numberofinstances"]');
            const parallelizationcriteria = configuratorElement.querySelector('select[name="parallelizationcriteria"]');

            if (groupBy.value !== "EntityType"){
                console.log("Validing the EntityStream configuration since groupBy is not EntityType")
                parallelization.checked = false
                numberofinstances.value = 1
                parallelizationcriteria.value = 'NA'
            } else {
                if (!parallelization.checked){
                    numberofinstances.value = 1
                    parallelizationcriteria.value = 'NA'
                }

            }
        }   

        function validateEntityStreamConfigurator(node) {
            const saveBtn = node.querySelector("button.save");

            if (saveBtn) {
                saveBtn.addEventListener("click", () => {
                    validateEntityStreamEntries(node);
                },true// <- capture phase, before the other listener
            );
            }
        }


        function toggleHidingField(configuratorElement, fieldSelector, triggerSelector, valueTrigger) {
            const fieldToToggle = configuratorElement.querySelector(fieldSelector);
            const triggerField = configuratorElement.querySelector(triggerSelector);

            if (fieldToToggle && triggerField) {
                // Find label text (two siblings before field) and the <br> nodes before/after
                const labelTextNode = fieldToToggle.previousSibling.previousSibling;
                const brBefore = labelTextNode?.previousSibling?.nodeName === "BR" ? labelTextNode.previousSibling : null;
                const brAfter = fieldToToggle.previousSibling?.nodeName === "BR" ? fieldToToggle.previousSibling : null;

                // Create wrapper div
                const wrapper = document.createElement("div");

                // Insert wrapper in place of first element (brBefore if exists, else labelTextNode)
                const insertBeforeNode = brBefore || labelTextNode;
                fieldToToggle.parentNode.insertBefore(wrapper, insertBeforeNode);

                // Move the elements inside the wrapper
                if (brBefore) brBefore.remove()
                if (labelTextNode) wrapper.appendChild(labelTextNode);
                if (brAfter) wrapper.appendChild(brAfter);
                wrapper.appendChild(fieldToToggle);

                // Hide by default
                wrapper.style.display = "none";

                // Listen for changes on trigger field
                triggerField.addEventListener('change', () => {
                    let match;
                    if (typeof valueTrigger === "boolean") {
                        // For checkboxes
                        match = (triggerField.checked === valueTrigger);
                    } else {
                        // For selects or text inputs
                        match = (triggerField.value === valueTrigger);
                    }
                    wrapper.style.display = match ? "" : "none";
                });
            }
        }

        function handleEntityStreamConfigurator(configuratorElement) {

            const groupBy = configuratorElement.querySelector('select[name="groupby"]');
            if (!groupBy) return; // Exit the function if not found because we are not in an EntityStream configuration

            // show the configuration of parallelization only when the parallelization check is checked
            toggleHidingField(configuratorElement,'input[name="numberofinstances"]', 'input[name="parallelization"]', true)
            toggleHidingField(configuratorElement,'select[name="parallelizationcriteria"]', 'input[name="parallelization"]', true)

            // hide the configuration of the parallelization part when groupBy is not EntityType
            toggleHidingField(configuratorElement,'input[name="parallelization"]', 'select[name="groupby"]', "EntityType")
            toggleHidingField(configuratorElement,'input[name="numberofinstances"]', 'select[name="groupby"]', "EntityType")
            toggleHidingField(configuratorElement,'select[name="parallelizationcriteria"]', 'select[name="groupby"]', "EntityType")

            // NGSI-LD delivery: only relevant when information model is NGSI-LD
            toggleHidingField(configuratorElement, 'select[name="ngsilddelivery"]', 'select[name="informationmodel"]', "NGSI-LD")
            toggleHidingField(configuratorElement, 'input[name="notificationpath"]', 'select[name="informationmodel"]', "NGSI-LD")

            const imSel = configuratorElement.querySelector('select[name="informationmodel"]');
            if (imSel) {
                setTimeout(function () {
                    imSel.dispatchEvent(new Event('change', { bubbles: true }));
                }, 0);
            }
        }
    }

    function openFogFunctionEditor(fogfunction) {
        if (fogfunction && fogfunction.designboard) {
            CurrentScene = fogfunction.designboard;
            showFogFunctionEditor(false);

            var topology = fogfunction.topology;
            $('#serviceName').val(topology.name);
            $('#serviceDescription').val(topology.description);
        }
    }


    function queryOperatorList() {
        fetch('/operator').then(res => res.json()).then(operators => {
            Object.values(operators).forEach(operator => {
                operatorList.push(operator.name);
            })         
        });         
    }
    
    function queryEntityTypeList() {
        fetch('/info/type').then(res => res.json()).then(dtypes => {
            for (var i = 0; i < dtypes.length; i++) {
                eTypeList.push(dtypes[i]);
            }            
        }); 
    }    
    

    function boardScene2FogFunction(scene) {
        // step 1: construct the service topology object   
        var attribute = {}    
        var topologyName = $('#serviceName').val();
        var serviceDescription = $('#serviceDescription').val();

        var topology = {};
        topology.name = topologyName;
        topology.description = serviceDescription;
        topology.tasks = generateTaskList(scene);

        // step 2: construct an intent object
        var intent = {};

        var uid = uuid();
        var sid = 'ServiceIntent.' + uid;
                    
        intent.id = sid;        
        intent.topology = topologyName;
        intent.priority = {
            'exclusive': false,
            'level': 0
        };
        intent.qos = "default";
        intent.geoscope = {
            "scopeType": "global",
            "scopeValue": "global"
        };
       
        var fogfunction = {};        
        fogfunction.name = topologyName;
        fogfunction.topology = topology;
        fogfunction.intent = intent;
        fogfunction.designboard = scene;
        fogfunction.status = 'enabled';
                       
        if (topologyName == '' || topology.name == '' || topology.tasks.length==0 || topology.tasks[0].operator == 'null' || 
        topology.tasks[0].operator == '' || topology.tasks[0].input_streams.length==0){
            alert('please provide the required inputs');
            return;
        }
        
        submitFogFunction(fogfunction);
    }

    function submitFogFunction(functionCtxObj) {
        console.log([functionCtxObj]);
        
        fetch("/fogfunction", {
            method: "POST",
            headers: {
                Accept: "application/json",
                "Content-Type": "application/json"
            },
            body: JSON.stringify([functionCtxObj])
        })
        .then(response => {
            console.log("submit a new fog function: ", response.status)
            showFogFunctions();
        })
        .catch(err => console.log(err));  
    }


    function generateTaskList(scene) {
        var tasklist = [];

        for (var i = 0; i < scene.blocks.length; i++) {
            var block = scene.blocks[i];
            if (block.type == 'Task') {
                var task = {};

                task.name = block.values['name'];
                task.operator = block.values['operator'];

                task.input_streams = [];
                task.output_streams = [];

                // look for all input streams associated with this task
                task.input_streams = findInputStream(scene, block.id);

                // figure out the defined output stream types                        
                for (var j = 0; j < block.values['outputs'].length; j++) {
                    var outputstream = {};
                    outputstream.entity_type = block.values['outputs'][j];
                    task.output_streams.push(outputstream);
                }

                tasklist.push(task);
            }
        }

        return tasklist;
    }

    function findInputStream(scene, blockid) {
        var inputstreams = [];

        for (var i = 0; i < scene.edges.length; i++) {
            var edge = scene.edges[i];
            if (edge.block2 == blockid) {
                var inputblockId = edge.block1;

                for (var j = 0; j < scene.blocks.length; j++) {
                    var block = scene.blocks[j];
                    if (block.id == inputblockId) {
                        if (block.type == 'Shuffle') {
                            var inputstream = {};

                            inputstream.selected_type = findInputType(scene, block.id)

                            if (block.values['selectedattributes'].length == 1 && block.values['selectedattributes'][0].toUpperCase() == 'ALL') {
                                inputstream.selected_attributes = [];
                            } else {
                                inputstream.selected_attributes = block.values['selectedattributes'];
                            }

                            inputstream.groupby = block.values['groupby'];
                            inputstream.scoped = true;
                            inputstream.information_model = block.values['informationmodel'];
                            if (block.values['informationmodel'] === 'NGSI-LD') {
                                var ldDelS = block.values['ngsilddelivery'] || 'upsert';
                                if (ldDelS === 'notification') {
                                    inputstream.ngsi_ld_delivery = 'notification';
                                }
                                var npS = (block.values['notificationpath'] || '').toString().trim();
                                if (npS) {
                                    inputstream.notification_path = npS;
                                }
                            }

                            inputstreams.push(inputstream)
                        } else if (block.type == 'EntityStream') {
                            var inputstream = {};

                            inputstream.selected_type = block.values['selectedtype'];

                            if (block.values['selectedattributes'].length == 1 && block.values['selectedattributes'][0].toUpperCase() == 'ALL') {
                                inputstream.selected_attributes = [];
                            } else {
                                inputstream.selected_attributes = block.values['selectedattributes'];
                            }

                            inputstream.groupby = block.values['groupby'];
                            inputstream.scoped = block.values['scoped'];
                            inputstream.information_model = block.values['informationmodel'];
                            if (block.values['informationmodel'] === 'NGSI-LD') {
                                var ldDel = block.values['ngsilddelivery'] || 'upsert';
                                if (ldDel === 'notification') {
                                    inputstream.ngsi_ld_delivery = 'notification';
                                }
                                var np = (block.values['notificationpath'] || '').toString().trim();
                                if (np) {
                                    inputstream.notification_path = np;
                                }
                            }

                            inputstreams.push(inputstream)
                        }
                    }
                }
            }
        }

        return inputstreams;
    }

    function findInputType(scene, blockId) {
        var inputType = "unknown";

        for (var i = 0; i < scene.edges.length; i++) {
            var edge = scene.edges[i];

            if (edge.block2 == blockId) {
                var index = edge.connector1[2];

                for (var j = 0; j < scene.blocks.length; j++) {
                    var block = scene.blocks[j];
                    if (block.id == edge.block1) {
                        inputType = block.values.outputs[index];
                    }
                }
            }
        }

        return inputType;
    }

    function showFogFunctions() {
        $('#info').html('list of all registered fog functions');

        var html = '<div style="margin-bottom: 10px;"><button id="registerFunction" type="button" class="btn btn-primary">register</button></div>';
        html += '<div id="functionList"></div>';

        $('#content').html(html);

        $("#registerFunction").click(function() {
            selectedFogFunction = null;
            CurrentScene = null
            showFogFunctionEditor(true);
        });

        // update the list of submitted fog functions
        updateFogFunctionList();
    }

    function updateFogFunctionList() {
        fetch('/fogfunction').then(res => res.json()).then(fogfunctions => {
            var fogfunctionList = Object.values(fogfunctions);
            displayFunctionList(fogfunctionList);             
        }).catch(function(error) {
            console.log(error);
            console.log('failed to fetch the list of fog functions');
        });
    }

    function displayFunctionList(fogFunctions) {
        if (fogFunctions == null || fogFunctions.length == 0) {
            return
        }

        var html = '<table class="table table-striped table-bordered table-condensed">';

        html += '<thead><tr>';
        html += '<th>Name</th>';
        html += '<th class="singlecolumn">Action</th>';
        html += '<th>Topology</th>';
        html += '<th>Intent</th>';
        html += '</tr></thead>';

        for (var i = 0; i < fogFunctions.length; i++) {
            var fogfunction = fogFunctions[i];
            
            html += '<td>' + fogfunction.name + '</td>';

            html += '<td>';
            
            html += '<button id="task-' + fogfunction.name + '" type="button" class="btn btn-primary btn-separator">tasks</button>';
            
            html += '<button id="editor-' + fogfunction.name + '" type="button" class="btn btn-primary btn-separator">view</button>';
            html += '<button id="delete-' + fogfunction.name + '" type="button" class="btn btn-primary btn-separator">delete</button>';
            
            if (fogfunction.status == 'enabled') {
                html += '<button id="status-' + fogfunction.name + '" type="button" class="btn btn-secondary btn-separator">disable</button>';                
            } else {
                html += '<button id="status-' + fogfunction.name + '" type="button" class="btn btn-success btn-separator">enable</button>';                
            }
            
            html += '</td>';

            html += '<td>' + fogfunction.topology.name + '</td>';

            html += '<td>' + JSON.stringify(fogfunction.intent) + '</td>';

            html += '</tr>';
        }

        html += '</table>';

        $('#functionList').html(html);

        // associate a click handler to the editor button
        for (var i = 0; i < fogFunctions.length; i++) {
            var fogfunction = fogFunctions[i];            
            // association handlers to the buttons
            var editorButton = document.getElementById('editor-' + fogfunction.name);
            editorButton.onclick = function(myFogFunction) {
                return function() {
                    console.log("editor buttion ",myFogFunction);
                    selectedFogFunction = myFogFunction;
                    openFogFunctionEditor(myFogFunction);
                };
            }(fogfunction);

            var deleteButton = document.getElementById('delete-' + fogfunction.name);
            deleteButton.onclick = function(myFogFunction) {
                return function() {
                    console.log("delete buttion ", myFogFunction);
                    deleteFogFunction(myFogFunction);
                };
            }(fogfunction);
            
            var statusButton = document.getElementById('status-' + fogfunction.name);
            statusButton.onclick = function(myFogFunction) {
                return function() {
                    console.log(statusButton.innerHTML);
                    if (this.innerHTML == "enable") {
                        enableFogFunction(myFogFunction);                        
                    } else {
                        disableFogFunction(myFogFunction);                                                
                    }
                };
            }(fogfunction);
            
            var taskButton = document.getElementById('task-' + fogfunction.name);
            taskButton.onclick = function(myFogFunction) {
                return function() {
                    console.log("taskButton buttion ",myFogFunction);
                    selectedFogFunction = myFogFunction;
                    getTaskByFogFunction(myFogFunction);
                };
            }(fogfunction);                        
        }
    }

    function deleteFogFunction(fogfunction) {                
        fetch("/fogfunction/" + fogfunction.name, {
            method: "DELETE"
        })
        .then(response => {
            console.log("delete a fog function: ", response.status)
            showFogFunctions();
        })
        .catch(err => console.log(err));   
    }
    
    function enableFogFunction(fogfunction) {                
        fetch("/fogfunction/" + fogfunction.name + "/enable")
        .then(response => {
            console.log("enable a fog function: ", response.status)
            showFogFunctions();
        })
        .catch(err => console.log(err));   
    }    
    
    function disableFogFunction(fogfunction) {                
        fetch("/fogfunction/" + fogfunction.name + "/disable")
        .then(response => {
            console.log("disable a fog function: ", response.status)
            showFogFunctions();
        })
        .catch(err => console.log(err));   
    }      

    function getTaskByFogFunction(fogfunction) {              
        fetch("/info/task/" + fogfunction.intent.id).then(res => res.json()).then(tasks => {
            displayTaskList(tasks);
        })
        .catch(err => console.log(err));   
    } 

    function uuid() {
        var uuid = "",
            i, random;
        for (i = 0; i < 32; i++) {
            random = Math.random() * 16 | 0;
            if (i == 8 || i == 12 || i == 16 || i == 20) {
                uuid += "-"
            }
            uuid += (i == 12 ? 4 : (i == 16 ? (random & 3 | 8) : random)).toString(16);
        }

        return uuid;
    }

    function displayTaskList(tasks) {
        if (tasks == null || tasks.length == 0) {
            $('#content').html('');
            return
        }

        var html = '<table class="table table-striped table-bordered table-condensed">';

        html += '<thead><tr>';
        html += '<th>ID</th>';
        html += '<th>Service</th>';
        html += '<th>Task</th>';
        html += '<th>Worker</th>';
        html += '<th>Status</th>';
        html += '</tr></thead>';

        for (var i = 0; i < tasks.length; i++) {
            var task = tasks[i];

            html += '<tr>';
            html += '<td>' + task.TaskID + '</td>';
            html += '<td>' + task.TopologyName + '</td>';
            html += '<td>' + task.TaskName + '</td>';
            html += '<td>' + task.Worker + '</td>';            

            if (task.Status == "paused") {
                html += '<td><font color="red">' + task.Status + '</font></td>';
            } else {
                html += '<td><font color="green">' + task.Status + '</font></td>';
            }

            html += '</tr>';
        }

        html += '</table>';

        $('#content').html(html);
    }


});