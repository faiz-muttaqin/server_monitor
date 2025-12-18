function jsonToHtml(jsonData) {
	var container = document.getElementById("mainCourseContent");

	function createHtmlElement(tag, attributes, content) {
		var element = document.createElement(tag);

		// Set attributes
		for (var key in attributes) {
			if (key === "tag" || key === "child") {
				continue;
			}
			if (attributes.hasOwnProperty(key)) {
				element.setAttribute(key, attributes[key]);
			}
		}

		// Set content
		if (content instanceof Array) {
			content.forEach(function (child) {
				var childElement = createHtmlElement(child.tag, child, child.child);
				element.appendChild(childElement);
			});
		} else if (typeof content === "string") {
			const parser = new DOMParser();
			try {
				const doc = parser.parseFromString(content, "text/html");
				if (
					Array.from(doc.body.childNodes).some((node) => node.nodeType === 1)
				) {
					element.innerHTML += content;
				} else {
					element.appendChild(document.createTextNode(content));
				}
			} catch (error) {
				element.appendChild(document.createTextNode(content));
			}
		}

		return element;
	}

	var path = window.location.pathname;

	// Extract the relevant parts from the path
	var parts = path.split("/").filter((part) => part !== "");

	// Convert the parts into the desired format
	var stringId = parts
		.map((part, index) => {
			if (index % 2 === 0) {
				// Even indices are section names (e.g., "courses", "modules", "contents")
				return part.substring(0, 2);
			} else {
				// Odd indices are section IDs (e.g., "1", "1", "3")
				return `-${part}-`;
			}
		})
		.join("");

	var i = 0;
	jsonData.forEach(function (item) {
		if (document.getElementById("userLevel").value === "true") {
			const viewBtn = document.getElementById("viewBtn");
			viewBtn.classList.remove("d-none");
			viewBtn.classList.add("d-flex");

			var validId = stringId + i;

			// Create a container div to wrap the buttons
			var buttonContainer = document.createElement("div");

			// Create the edit button
			var editButton = document.createElement("button");
			editButton.id = "change-" + validId;
			editButton.className = "btn text-primary edit-btn px-1";
			editButton.innerHTML = 'Edit <i class="bx bx-edit"></i>';
			editButton.addEventListener("click", function () {
				editContents(validId);
			});

			// Create the delete button
			var deleteButton = document.createElement("button");
			deleteButton.id = "delete-" + validId;
			deleteButton.className = "btn text-danger edit-btn  px-1";
			deleteButton.innerHTML = '<i class="bx bx-trash" ></i>';
			deleteButton.addEventListener("click", function () {
				deleteContents(validId);
			});

			// Append both buttons to the container div
			buttonContainer.className = "align-self-end d-flex flex-row";
			buttonContainer.appendChild(editButton);
			buttonContainer.appendChild(deleteButton);

			// Append the container div to your main container
			container.appendChild(buttonContainer);
		}

		var element = createHtmlElement(item.tag, item, item.child);
		element.id = validId;
		container.appendChild(element);
		i++;
	});

	//Content Button
	var newContentButton = document.createElement("a");
	newContentButton.className =
		"btn d-flex flex-column align-items-center justify-content-center text-primary";
	newContentButton.innerHTML =
		'<i class="bx bx-plus-circle bx-lg"></i>Sub Content';
	newContentButton.addEventListener("click", function () {
		i++;
		var contentID = stringId + i;

		if (document.getElementById("userLevel").value === "true") {
			// Create a container div to wrap the buttons
			var buttonContainer = document.createElement("div");

			// Create the edit button
			var editButton = document.createElement("button");
			editButton.id = "change-" + contentID;
			editButton.className = "btn text-primary edit-btn px-1";
			editButton.innerHTML = 'Save <i class="bx bx-save"></i>';
			editButton.addEventListener("click", function () {
				editContents(contentID);
			});

			// Create the delete button
			var deleteButton = document.createElement("button");
			deleteButton.id = "delete-" + contentID;
			deleteButton.className = "btn text-danger edit-btn  px-1";
			deleteButton.innerHTML = '<i class="bx bx-trash" ></i>';
			deleteButton.addEventListener("click", function () {
				var elementToRemove = document.getElementById(contentID); // Replace with your actual element ID

				if (elementToRemove) {
					var siblingsToRemove = [
						elementToRemove.previousElementSibling,
						elementToRemove.previousElementSibling.previousElementSibling,
					];

					// Remove the element and its two above siblings
					elementToRemove.remove();
					siblingsToRemove.forEach(function (sibling) {
						if (sibling) {
							sibling.remove();
						}
					});
				}
			});

			// Append both buttons to the container div
			buttonContainer.className = "align-self-end d-flex flex-row";
			buttonContainer.appendChild(editButton);
			buttonContainer.appendChild(deleteButton);

			// Append the container div to your main container
			container.appendChild(buttonContainer);

			//create New div
			var div = document.createElement("div");
			div.id = contentID;
			container.appendChild(div);
		}
		const fullToolbar = [
			[
				{
					font: [],
				},
				{
					size: [],
				},
			],
			["bold", "italic", "underline", "strike"],
			[
				{
					color: [],
				},
				{
					background: [],
				},
			],
			[
				{
					script: "super",
				},
				{
					script: "sub",
				},
			],
			[
				{
					header: "1",
				},
				{
					header: "2",
				},
				"blockquote",
				"code-block",
			],
			[
				{
					list: "ordered",
				},
				{
					list: "bullet",
				},
				{
					indent: "-1",
				},
				{
					indent: "+1",
				},
			],
			[{ direction: "rtl" }],
			["link", "image", "video", "formula"],
			["clean"],
		];
		const fullEditor = new Quill("#" + contentID, {
			bounds: "#" + contentID,
			placeholder: "Type Something...",
			modules: {
				formula: true,
				toolbar: fullToolbar,
			},
			theme: "snow",
		});
	});
	var newVideoButton = document.createElement("a");
	newVideoButton.className =
		"btn d-flex flex-column align-items-center justify-content-center text-primary";
	newVideoButton.innerHTML = '<i class="bx bx-video-plus bx-lg"></i>Video';
	newVideoButton.addEventListener("click", function () {
		i++;
		var contentID = stringId + i;

		if (document.getElementById("userLevel").value === "true") {
			// Create a container div to wrap the buttons
			var buttonContainer = document.createElement("div");

			// Create the edit button
			var editButton = document.createElement("button");
			editButton.id = "change-" + contentID;
			editButton.className = "btn text-primary edit-btn px-1";
			editButton.innerHTML = 'Save <i class="bx bx-save"></i>';
			editButton.addEventListener("click", function () {
				editContents(contentID, "video", true);
			});

			// Create the delete button
			var deleteButton = document.createElement("button");
			deleteButton.id = "delete-" + contentID;
			deleteButton.className = "btn text-danger edit-btn  px-1";
			deleteButton.innerHTML = '<i class="bx bx-trash" ></i>';
			deleteButton.addEventListener("click", function () {
				var elementToRemove = document.getElementById(contentID); // Replace with your actual element ID

				if (elementToRemove) {
					var siblingsToRemove = [elementToRemove.previousElementSibling];

					// Remove the element and its two above siblings
					elementToRemove.remove();
					siblingsToRemove.forEach(function (sibling) {
						if (sibling) {
							sibling.remove();
						}
					});
				}
			});

			// Append both buttons to the container div
			buttonContainer.className = "align-self-end d-flex flex-row";
			buttonContainer.appendChild(editButton);
			buttonContainer.appendChild(deleteButton);

			// Append the container div to your main container
			container.appendChild(buttonContainer);

			//create New div _________________________________________________________
			// Create the main container div
			var wrapperDiv = document.createElement("div");
			wrapperDiv.id = contentID;
			wrapperDiv.className = "wrapper-ua";

			// Create the form element
			var formElement = document.createElement("form");
			formElement.id = "form-" + contentID;
			formElement.className = "form-ua text-primary";
			formElement.action = "#";

			// Create the file input element
			var fileInput = document.createElement("input");
			fileInput.className = "file-input";
			fileInput.type = "file";
			fileInput.name = "file";
			fileInput.hidden = true;

			// Create the cloud upload icon
			var cloudUploadIcon = document.createElement("i");
			cloudUploadIcon.className = "bx bx-cloud-upload text-primary";

			// Create the paragraph element
			var paragraphElement = document.createElement("p");
			paragraphElement.className = "m-0";
			paragraphElement.textContent = "Click To Upload Video";

			// Append elements to the form element
			formElement.appendChild(fileInput);
			formElement.appendChild(cloudUploadIcon);
			formElement.appendChild(paragraphElement);

			// Create the progress area section
			var progressArea = document.createElement("div");
			progressArea.className = "sec-ua progress-area";

			// Create the uploaded area section
			var uploadedArea = document.createElement("div");
			uploadedArea.className = "sec-ua uploaded-area";

			var uploadedArea = document.createElement("div");
			uploadedArea.className = "sec-ua uploaded-area";

			// Create the card div
			var cardDiv = document.createElement("div");
			cardDiv.innerHTML = `<div class="card-datatable table-responsive">
                <table class="list-video-edit table border-top">
                  <thead>
                    <tr>
                      <th></th>
                      <th>id</th>
                      <th>Name</th>
                      <th>Action</th>
                    </tr>
                  </thead>
                </table>
              </div>
            `;

			// Append all elements to the main container div
			wrapperDiv.appendChild(formElement);
			wrapperDiv.appendChild(progressArea);
			wrapperDiv.appendChild(uploadedArea);
			// Append the card div to the wrapperDiv
			wrapperDiv.appendChild(cardDiv);

			// Append the main container div to the document body (or any other parent element)
			container.appendChild(wrapperDiv);

			const form_uf = document.querySelector("#form-" + contentID);
			if (form_uf) {
				const fileInput = document.querySelector(".file-input");
				const progressArea = document.querySelector(".progress-area");
				const uploadedArea = document.querySelector(".uploaded-area");
				// form click event
				form_uf.addEventListener("click", () => {
					fileInput.click();
				});

				fileInput.onchange = ({ target }) => {
					let file = target.files[0]; //getting file [0] this means if user has selected multiple files then get first one only
					if (file) {
						let fileName = file.name; //getting file name
						if (fileName.length >= 12) {
							//if file name length is greater than 12 then split it and add ...
							let splitName = fileName.split(".");
							fileName = splitName[0].substring(0, 13) + "... ." + splitName[1];
						}
						// if (!(file.name.endsWith('.zip'))) {
						//   Swal.fire('Invalid File, must be .zip', '', 'error');
						//   return;
						// }
						Swal.fire({
							title: "Upload Verification",
							text: `Are you sure you want to upload "${file.name}"?`,
							icon: "question",
							showCancelButton: true,
							confirmButtonText: "Upload!",
							cancelButtonText: "Cancel",
							confirmButtonClass: "btn btn-primary",
							cancelButtonClass: "btn btn-outline-secondary ml-1",
							buttonsStyling: false,
						}).then((result) => {
							if (result.value) {
								// User clicked "Upload"
								uploadFile(file.name); // Replace with your upload function
							} else if (result.dismiss === Swal.DismissReason.cancel) {
								// User clicked "Cancel" or outside the modal
								Swal.fire(
									"Cancelled",
									"The upload process was cancelled",
									"info"
								);
							}
						});
					}
				};
				// file upload function
				function uploadFile(name) {
					var contentValidatorElement =
						document.getElementById("contentValidator").value;
					let xhr = new XMLHttpRequest(); //creating new xhr object (AJAX)
					xhr.open("POST", "/upload/video/" + contentValidatorElement); //sending post request to the specified URL
					xhr.upload.addEventListener("progress", ({ loaded, total }) => {
						//file uploading progress event
						let fileLoaded = Math.floor((loaded / total) * 100); //getting percentage of loaded file size
						let fileTotal = Math.floor(total / 1000); //gettting total file size in KB from bytes
						let fileSize;
						// if file size is less than 1024 then add only KB else convert this KB into MB
						fileTotal < 1024
							? (fileSize = fileTotal + " KB")
							: (fileSize = (loaded / (1024 * 1024)).toFixed(2) + " MB");
						let progressHTML = `<li class="row px-2">
                              <i class="fas fa-file-alt"></i>
                              <div class="content">
                                <div class="details">
                                <span class="name">${name} • Uploading</span>
                                <span class="percent">${fileLoaded}%</span>
                                </div>
                                <div class="progress-bar">
                                <div class="progress" style="width: ${fileLoaded}%"></div>
                                </div>
                              </div>
                              </li>`;
						// uploadedArea.innerHTML = ""; //uncomment this line if you don't want to show push history
						uploadedArea.classList.add("onprogress");
						progressArea.innerHTML = progressHTML;
						if (loaded == total) {
							progressArea.innerHTML = "";
							let uploadedHTML = `<li class="row px-2">
                                <div class="content upload">
                                <i class="fas fa-file-alt"></i>
                                <div class="details">
                                  <span class="name">${name} • Uploaded</span>
                                  <span class="size">${fileSize}</span>
                                </div>
                                </div>
                                <i class="fas fa-check"></i>
                              </li>`;
							uploadedArea.classList.remove("onprogress");
							// uploadedArea.innerHTML = uploadedHTML; //uncomment this line if you don't want to show push history
							uploadedArea.insertAdjacentHTML("afterbegin", uploadedHTML); //remove this line if you don't want to show push history
						}
					});
					let data = new FormData(form_uf); //FormData is an object to easily send form data
					xhr.onload = function () {
						if (xhr.status === 200) {
							// Successful response from PHP
							const response = JSON.parse(xhr.responseText);
						} else {
							console.error("Error:", xhr.status);
						}
					};
					xhr.send(data); //sending form data
				}
			}
			var dt_basic_table = $(".list-video-edit");
			if (dt_basic_table.length) {
				dt_basic = dt_basic_table.DataTable({
					ajax: "/table/video",
					columns: [
						{ data: "" },
						{ data: "id" },
						{ data: "filename" },
						{ data: "" },
					],
					columnDefs: [
						{
							// For Responsive
							className: "control",
							orderable: false,
							searchable: false,
							responsivePriority: 2,
							targets: 0,
							render: function (data, type, full, meta) {
								return "";
							},
						},
						{
							targets: 1,
							searchable: false,
							visible: false,
						},
						{
							// Avatar image/badge, Name and post
							targets: 2,
							responsivePriority: 4,
							render: function (data, type, full, meta) {
								var $user_src = full["path"],
									$id = full["id"],
									$name = full["filename"],
									$post = full["updated_by"],
									$type = full["type"];
								$thumbnail = full["thumbnail"];
								if ($user_src) {
									if ($type === "video") {
										// For Avatar image
										var $output =
											'<video controls="" id="example-plyr-video-player-' +
											$id +
											'" playsinline="" poster="' +
											$thumbnail +
											'" width="" class="w-100 round" style="max-width:300px;"><source src="' +
											$user_src +
											'" type="video/mp4"></video>';
									} else {
										var $output =
											'<img src="' +
											assetsPath +
											"img/avatars/" +
											$user_src +
											'" alt="Avatar" class="rounded-circle">';
									}
								} else {
									// For Avatar badge
									var stateNum = Math.floor(Math.random() * 6);
									var states = [
										"success",
										"danger",
										"warning",
										"info",
										"dark",
										"primary",
										"secondary",
									];
									var $state = states[stateNum],
										$name = full["filename"],
										$initials = $name.match(/\b\w/g) || [];
									$initials = (
										($initials.shift() || "") + ($initials.pop() || "")
									).toUpperCase();
									$output =
										'<span class="avatar-initial rounded-circle bg-label-' +
										$state +
										'">' +
										$initials +
										"</span>";
								}
								// Creates full output for row
								var $row_output =
									'<div class="d-flex justify-content-start align-items-center user-name">' +
									'<div class="d-flex flex-column">' +
									'<span class="emp_name text-truncate">' +
									$name +
									"</span>" +
									'<small class="emp_post text-truncate text-muted">' +
									$post +
									"</small>" +
									"</div>" +
									"</div>";
								return $row_output;
							},
						},
						{
							// Actions
							targets: -1,
							title: "Actions",
							orderable: false,
							searchable: false,
							render: function (data, type, full, meta) {
								var $user_src = full["path"];
								return (
									'<div class="d-inline-block">' +
									'<a href="javascript:;" class="btn btn-sm btn-icon dropdown-toggle hide-arrow" data-bs-toggle="dropdown"><i class="bx bx-dots-vertical-rounded"></i></a>' +
									'<ul class="dropdown-menu dropdown-menu-end m-0">' +
									'<li><a href="javascript:;" class="dropdown-item">Details</a></li>' +
									'<li><a href="javascript:;" class="dropdown-item">Archive</a></li>' +
									'<div class="dropdown-divider"></div>' +
									'<li><a href="javascript:;" class="dropdown-item text-danger delete-record">Delete</a></li>' +
									"</ul>" +
									"</div>" +
									"<a onclick=\"appendVideoFirst('" +
									wrapperDiv.id +
									"', `" +
									$user_src +
									'` )" class="btn btn-sm btn-icon item-edit"><i class="bx bxs-video-plus"></i></a>'
								);
							},
						},
					],
					order: [[1, "desc"]],
					dom: '<"card-header flex-column flex-md-row"<"head-label text-center"><"dt-action-buttons text-end pt-3 pt-md-0"B>><"row"<"col-sm-12 col-md-6"l><"col-sm-12 col-md-6 d-flex justify-content-center justify-content-md-end"f>>t<"row"<"col-sm-12 col-md-6"i><"col-sm-12 col-md-6"p>>',
					displayLength: 7,
					lengthMenu: [7, 10, 25, 50, 75, 100],
					buttons: [],
					responsive: {
						details: {
							display: $.fn.dataTable.Responsive.display.modal({
								header: function (row) {
									var data = row.data();
									return "Details of " + data["filename"];
								},
							}),
							type: "column",
							renderer: function (api, rowIdx, columns) {
								var data = $.map(columns, function (col, i) {
									return col.title !== "" // ? Do not show row in modal popup if title is blank (for check box)
										? '<tr data-dt-row="' +
												col.rowIndex +
												'" data-dt-column="' +
												col.columnIndex +
												'">' +
												"<td>" +
												col.title +
												":" +
												"</td> " +
												"<td>" +
												col.data +
												"</td>" +
												"</tr>"
										: "";
								}).join("");

								return data
									? $('<table class="table"/><tbody />').append(data)
									: false;
							},
						},
					},
				});
				$("div.card-header.flex-column").addClass("d-none");
				dt_basic_table.find("thead").addClass("d-none");
			}
		}
	});
	var newPdfButton = document.createElement("a");
	newPdfButton.className =
		"btn d-flex flex-column align-items-center justify-content-center text-primary";
	newPdfButton.innerHTML = '<i class="bx bxs-file-plus bx-lg" ></i>Pdf';
	newPdfButton.addEventListener("click", function () {
		i++;
		var contentID = stringId + i;

		if (document.getElementById("userLevel").value === "true") {
			// Create a container div to wrap the buttons
			var buttonContainer = document.createElement("div");

			// Create the edit button
			var editButton = document.createElement("button");
			editButton.id = "change-" + contentID;
			editButton.className = "btn text-primary edit-btn px-1";
			editButton.innerHTML = 'Save <i class="bx bx-save"></i>';
			editButton.addEventListener("click", function () {
				editContents(contentID);
			});

			// Create the delete button
			var deleteButton = document.createElement("button");
			deleteButton.id = "delete-" + contentID;
			deleteButton.className = "btn text-danger edit-btn  px-1";
			deleteButton.innerHTML = '<i class="bx bx-trash" ></i>';
			deleteButton.addEventListener("click", function () {
				var elementToRemove = document.getElementById(contentID); // Replace with your actual element ID

				if (elementToRemove) {
					var siblingsToRemove = [
						elementToRemove.previousElementSibling,
						elementToRemove.previousElementSibling.previousElementSibling,
					];

					// Remove the element and its two above siblings
					elementToRemove.remove();
					siblingsToRemove.forEach(function (sibling) {
						if (sibling) {
							sibling.remove();
						}
					});
				}
			});

			// Append both buttons to the container div
			buttonContainer.className = "align-self-end d-flex flex-row";
			buttonContainer.appendChild(editButton);
			buttonContainer.appendChild(deleteButton);

			// Append the container div to your main container
			container.appendChild(buttonContainer);

			//create New div
			var div = document.createElement("div");
			div.id = contentID;
			container.appendChild(div);
		}
		const fullToolbar = [
			[
				{
					font: [],
				},
				{
					size: [],
				},
			],
			["bold", "italic", "underline", "strike"],
			[
				{
					color: [],
				},
				{
					background: [],
				},
			],
			[
				{
					script: "super",
				},
				{
					script: "sub",
				},
			],
			[
				{
					header: "1",
				},
				{
					header: "2",
				},
				"blockquote",
				"code-block",
			],
			[
				{
					list: "ordered",
				},
				{
					list: "bullet",
				},
				{
					indent: "-1",
				},
				{
					indent: "+1",
				},
			],
			[{ direction: "rtl" }],
			["link", "image", "video", "formula"],
			["clean"],
		];
		const fullEditor = new Quill("#" + contentID, {
			bounds: "#" + contentID,
			placeholder: "Type Something...",
			modules: {
				formula: true,
				toolbar: fullToolbar,
			},
			theme: "snow",
		});
	});
	var newQuizButton = document.createElement("a");
	newQuizButton.className =
		"btn d-flex flex-column align-items-center justify-content-center text-primary";
	newQuizButton.innerHTML = '<i class="bx bx-message-add bx-lg"></i>Quiz';
	newQuizButton.addEventListener("click", function () {
		i++;
		var contentID = stringId + i;

		if (document.getElementById("userLevel").value === "true") {
			// Create a container div to wrap the buttons
			var buttonContainer = document.createElement("div");

			// Create the edit button
			var editButton = document.createElement("button");
			editButton.id = "change-" + contentID;
			editButton.className = "btn text-primary edit-btn px-1";
			editButton.innerHTML = 'Save <i class="bx bx-save"></i>';
			editButton.addEventListener("click", function () {
				editContents(contentID);
			});

			// Create the delete button
			var deleteButton = document.createElement("button");
			deleteButton.id = "delete-" + contentID;
			deleteButton.className = "btn text-danger edit-btn  px-1";
			deleteButton.innerHTML = '<i class="bx bx-trash" ></i>';
			deleteButton.addEventListener("click", function () {
				var elementToRemove = document.getElementById(contentID); // Replace with your actual element ID

				if (elementToRemove) {
					var siblingsToRemove = [
						elementToRemove.previousElementSibling,
						elementToRemove.previousElementSibling.previousElementSibling,
					];

					// Remove the element and its two above siblings
					elementToRemove.remove();
					siblingsToRemove.forEach(function (sibling) {
						if (sibling) {
							sibling.remove();
						}
					});
				}
			});

			// Append both buttons to the container div
			buttonContainer.className = "align-self-end d-flex flex-row";
			buttonContainer.appendChild(editButton);
			buttonContainer.appendChild(deleteButton);

			// Append the container div to your main container
			container.appendChild(buttonContainer);

			//create New div
			var div = document.createElement("div");
			div.id = contentID;
			container.appendChild(div);
		}
		const fullToolbar = [
			[
				{
					font: [],
				},
				{
					size: [],
				},
			],
			["bold", "italic", "underline", "strike"],
			[
				{
					color: [],
				},
				{
					background: [],
				},
			],
			[
				{
					script: "super",
				},
				{
					script: "sub",
				},
			],
			[
				{
					header: "1",
				},
				{
					header: "2",
				},
				"blockquote",
				"code-block",
			],
			[
				{
					list: "ordered",
				},
				{
					list: "bullet",
				},
				{
					indent: "-1",
				},
				{
					indent: "+1",
				},
			],
			[{ direction: "rtl" }],
			["link", "image", "video", "formula"],
			["clean"],
		];
		const fullEditor = new Quill("#" + contentID, {
			bounds: "#" + contentID,
			placeholder: "Type Something...",
			modules: {
				formula: true,
				toolbar: fullToolbar,
			},
			theme: "snow",
		});
	});

	var newContainerButton = document.createElement("div");
	newContainerButton.className = "add-new-content-btn";
	newContainerButton.className =
		"d-flex flex-row border my-3 align-items-center justify-content-center text-primary edit-btn";
	newContainerButton.appendChild(newContentButton);
	newContainerButton.appendChild(newVideoButton);
	newContainerButton.appendChild(newPdfButton);
	newContainerButton.appendChild(newQuizButton);

	// Insert the newContentButton after the mainCourseContent element
	container.insertAdjacentElement("afterend", newContainerButton);
}

function appendVideoFirst(idWrapper, videoSrc) {
	var parentElement = $("#" + idWrapper);

	// Check if the first child is a div with class 'plyr'
	if (parentElement.children().first().is("div.plyr")) {
		// If it is, remove the div
		parentElement.children().first().remove();
	}

	// Add the videoHTML as the first child of the parent element
	var videoHTML =
		'<video controls="" id="video-player-' +
		idWrapper +
		'" playsinline="" width="" class="w-100" ><source src="' +
		videoSrc +
		'" type="video/mp4"></video>';
	parentElement.prepend(videoHTML);

	// Initialize Plyr for the newly added video element
	const videoElements = document.querySelectorAll("video");
	videoElements.forEach(function (video, index) {
		console.log(video, index);
		if (!video.id) {
			video.id = "video-player-" + (index + 1);
		}
		const videoPlayer = new Plyr("#" + video.id);
	});
}

var contentValidatorElement = document.getElementById("contentValidator");

if (contentValidatorElement) {
	var contentValue = contentValidatorElement.value;

	fetch("/contents/" + contentValue)
		.then(function (response) {
			if (!response.ok) {
				throw new Error("Network response was not ok");
			}
			return response.text();
		})
		.then(function (data) {
			try {
				var parsedData = JSON.parse(data);
				var jsonObject = JSON.parse(parsedData.data);
				// console.log(jsonObject);
				jsonToHtml(jsonObject);
				makeIframeResponsive("ql-video", 3, 2);

				const videoElements = document.querySelectorAll("video");
				videoElements.forEach(function (video, index) {
					// Check if the video element has an ID
					if (!video.id) {
						// If not, set a new ID
						video.id = "video-player-" + (index + 1);
					}
					const videoPlayer = new Plyr("#" + video.id);
				});
			} catch (error) {
				console.error("Error parsing JSON:", error);
			}
		})
		.catch(function (error) {
			console.log("Error fetching content:", error);
		});
}

function makeIframeResponsive(
	containerClass,
	aspectRatioWidth,
	aspectRatioHeight
) {
	// Get all elements with the specified class
	var iframes = document.getElementsByClassName(containerClass);

	// Function to update the height based on the width
	function updateIframeHeight(iframe) {
		iframe.style.width = "100%";
		var currentWidth = iframe.offsetWidth;
		var calculatedHeight =
			(currentWidth * aspectRatioHeight) / aspectRatioWidth;
		iframe.style.height = calculatedHeight + "px";

		// Prevent right-click on the iframe
		iframe.addEventListener("contextmenu", function (e) {
			e.preventDefault();
		});
		// Prevent long press on the iframe (for touchscreen devices)
		iframe.addEventListener("touchstart", function (e) {
			var now = new Date().getTime();
			var delta = now - (iframe.touchstart || now + 1);
			iframe.touchstart = now;
			if (delta < 500 && delta > 0) {
				e.preventDefault();
			}
		});
	}

	// Function to apply the updateIframeHeight function to all elements
	function updateAllIframesHeight() {
		for (var i = 0; i < iframes.length; i++) {
			updateIframeHeight(iframes[i]);
		}
	}

	// Call the function on initial page load
	updateAllIframesHeight();

	// Listen for window resize events to update the height dynamically
	window.addEventListener("resize", updateAllIframesHeight);
}

function editContents(contentID, type, isNewUpload) {
	const btnChange = document.getElementById("change-" + contentID);
	if (btnChange.innerHTML.includes("Edit")) {
		btnChange.innerHTML = 'Save <i class="bx bx-save"></i>';
		const fullToolbar = [
			[
				{
					font: [],
				},
				{
					size: [],
				},
			],
			["bold", "italic", "underline", "strike"],
			[
				{
					color: [],
				},
				{
					background: [],
				},
			],
			[
				{
					script: "super",
				},
				{
					script: "sub",
				},
			],
			[
				{
					header: "1",
				},
				{
					header: "2",
				},
				"blockquote",
				"code-block",
			],
			[
				{
					list: "ordered",
				},
				{
					list: "bullet",
				},
				{
					indent: "-1",
				},
				{
					indent: "+1",
				},
			],
			[{ direction: "rtl" }],
			["link", "image", "video", "formula"],
			["clean"],
		];
		const fullEditor = new Quill("#" + contentID, {
			bounds: "#" + contentID,
			placeholder: "Type Something...",
			modules: {
				formula: true,
				toolbar: fullToolbar,
			},
			theme: "snow",
		});
	} else {
		const parentElement = document.getElementById(contentID);
		const firstChild = parentElement.children[0];

		// Alternatively, if you want to check if the first child is a form
		if (firstChild.tagName.toLowerCase() === "form") {
			Swal.fire("Error!", "Please Choose Content", "error");
			return;
		}

		if (type === "video" && isNewUpload == true) {
			Swal.fire({
				title: "Do you want to Upload New Content?",
				icon: "warning",
				showCancelButton: true,
				confirmButtonText: "Save",
				cancelButtonText: "Cancel",
				reverseButtons: true,
			}).then((result) => {
				if (result.isConfirmed) {
					if (parentElement.children.length > 0) {
						const postData = {
							level: 0,
							data: firstChild.querySelector("video").outerHTML,
						};

						fetch("/contents/" + contentValue, {
							method: "POST",
							headers: {
								"Content-Type": "application/json", // or 'application/x-www-form-urlencoded' depending on your server expectations
								// Add any other headers if needed
							},
							// Convert the postData object to JSON format if sending JSON data
							body: JSON.stringify(postData),
						})
							.then(function (response) {
								if (!response.ok) {
									throw new Error("Network response was not ok");
								}
								return response.text();
							})
							.then(function (data) {
								// console.log(data);
								btnChange.innerHTML = 'Edit <i class="bx bx-edit"></i>';
								// Get all child elements except the first one
								var childElementsToRemove = Array.from(
									parentElement.children
								).slice(1);

								// Remove each child element
								childElementsToRemove.forEach(function (childElement) {
									parentElement.removeChild(childElement);
								});

								Swal.fire("Saved!", "Your changes have been saved.", "success");
							})
							.catch(function (error) {
								Swal.fire("Error!", "Your changes Failed to saved.", "error");
								console.log("Error fetching content:", error);
							});
					}
				}
			});
		} else {
			const parts = contentID.split("-");
			const lastNumber = Number(parts[parts.length - 1]);
			const parentElement = document.getElementById(contentID);

			Swal.fire({
				title: "Do you want to save changes?",
				icon: "warning",
				showCancelButton: true,
				confirmButtonText: "Save",
				cancelButtonText: "Cancel",
				reverseButtons: true,
			}).then((result) => {
				if (result.isConfirmed) {
					btnChange.innerHTML = 'Edit <i class="bx bx-edit"></i>';
					if (parentElement.previousElementSibling) {
						const previousSibling = parentElement.previousElementSibling;
						previousSibling.parentNode.removeChild(previousSibling);
					}

					if (parentElement.children.length > 0) {
						// Step 3: Get a reference to the first child
						const firstChild = parentElement.children[0];

						Array.from(firstChild.attributes).forEach((attribute) => {
							firstChild.removeAttribute(attribute.name);
						});
						firstChild.id = contentID;
						parentElement.parentNode.replaceChild(firstChild, parentElement);
						const postData = {
							level: lastNumber,
							data: firstChild.innerHTML,
						};
						console.log(firstChild.innerHTML);

						fetch("/contents/" + contentValue, {
							method: "PATCH",
							headers: {
								"Content-Type": "application/json", // or 'application/x-www-form-urlencoded' depending on your server expectations
								// Add any other headers if needed
							},
							// Convert the postData object to JSON format if sending JSON data
							body: JSON.stringify(postData),
						})
							.then(function (response) {
								if (!response.ok) {
									throw new Error("Network response was not ok");
								}
								return response.text();
							})
							.then(function (data) {
								console.log(data);
								Swal.fire("Saved!", "Your changes have been saved.", "success");
							})
							.catch(function (error) {
								Swal.fire("Error!", "Your changes Failed to saved.", "error");
								console.log("Error fetching content:", error);
							});
					}
				} else {
					// User clicked "Cancel" or closed the dialog
					// Swal.fire('Cancelled', 'Your changes have not been saved.', 'info');
				}
			});
		}
	}
}

function deleteContents(contentID) {
	Swal.fire({
		title: "ARE YOU SURE TO DELETE?",
		icon: "warning",
		showCancelButton: true,
		confirmButtonText: "I AM SURE",
		cancelButtonText: "Cancel",
		reverseButtons: true,
	}).then((result) => {
		if (result.isConfirmed) {
			const parts = contentID.split("-");
			const lastNumber = Number(parts[parts.length - 1]);
			const postData = {
				level: lastNumber,
			};

			fetch("/contents/" + contentValue, {
				method: "DELETE",
				headers: {
					"Content-Type": "application/json", // or 'application/x-www-form-urlencoded' depending on your server expectations
					// Add any other headers if needed
				},
				// Convert the postData object to JSON format if sending JSON data
				body: JSON.stringify(postData),
			})
				.then(function (response) {
					if (!response.ok) {
						throw new Error("Network response was not ok");
					}
					return response.text();
				})
				.then(function (data) {
					console.log(data);
					Swal.fire("Saved!", "Your content have been DELETED.", "success");
				})
				.catch(function (error) {
					Swal.fire("Error!", "Your content Failed to delete.", "error");
					console.log("Error fetching content:", error);
				});
		}
	});
}

function viewBtn() {
	document.querySelectorAll(".edit-btn").forEach(function (element) {
		element.classList.toggle("d-flex");
		element.classList.toggle("d-none");
	});
}
