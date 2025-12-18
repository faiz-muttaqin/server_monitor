package webgui

import (
	"fmt"
	"html/template"

	"github.com/sirupsen/logrus"
)

type ColumnCConfig struct {
	ClassName string
	Targets   int
	Visible   bool
	Orderable bool
	Render    template.HTML
}
type ColumnC struct {
	Data, Type, EditId                    string
	Header, Filter, EditForm, InsertField template.HTML
	ColumnCConfig                         ColumnCConfig
	Visible, Orderable, Filterable        bool
	Editable, Insertable                  bool
	Passwordable                          bool
	SelectableSrc                         template.URL
}

func ListCard(title, table_name, endpoint string, page_length int, length_menu []int, order []any, table_columnCs []ColumnC, insertable, editable, deletable, hideHeader, passwordable bool, scrollUpDown, scrollLeftRight bool, exportType []string) template.HTML {
	var columnC_array []int
	for i, col := range table_columnCs {
		if col.Visible {
			columnC_array = append(columnC_array, i)
		}
		table_columnCs[i].Filterable = true
		switch col.Type {
		case "string":

			filter_id := "ft_" + table_name + "_" + col.Data
			edit_id := "ed_" + table_name + "_" + col.Data
			insert_id := "in_" + table_name + "_" + col.Data
			table_columnCs[i].EditId = edit_id
			if table_columnCs[i].SelectableSrc != "" {
				table_columnCs[i].EditForm = template.HTML(fmt.Sprintf(`
				<label class="form-label">%s</label>
				<input
					id="%s"
					type="text"
					class="form-control"
					name="%s"
					data-columnC="%d"
					placeholder="%s Text"
					data-columnC-index="%d" />

				<script>
				fetch('%s')
					.then(response => response.json())
					.then(data => {
						var prefetchExample = new Bloodhound({
							datumTokenizer: Bloodhound.tokenizers.whitespace,
							queryTokenizer: Bloodhound.tokenizers.whitespace,
							local: data // Use fetched data directly as the suggestion source
						});

						// Function to render default suggestions or search results
						function renderDefaults(q, sync) {
							if (q === '') {
								sync(prefetchExample.all()); // Show all suggestions when the query is empty
							} else {
								prefetchExample.search(q, sync); // Search based on the query
							}
						}

						// Initialize Typeahead on the input field
						$('#%s').typeahead(
							{
								hint: true,
								highlight: true,
								minLength: 0
							},
							{
								name: 'options',
								source: renderDefaults
							}
						);

						// Show all options when the input is focused and empty
						$('#%s').on('focus', function() {
							if (this.value === '') {
								$(this).typeahead('val', ''); // Clear the input to trigger default suggestions
								$(this).typeahead('open'); // Open the dropdown with all suggestions
							}
						});
						// Trigger a function when an option is selected from the dropdown
						$('#%s').on('typeahead:select', function(ev, suggestion) {
							// Perform an action here, e.g., trigger a keyup event, call a function, etc.
							$(this).trigger('keyup'); // Example: Trigger the keyup event
							filterColumnC($(this).attr('data-columnC'), $(this).val()); // Example: Trigger your filtering function
						});
					})
					.catch(error => console.error('Error fetching options data:', error));
				</script>
				  `, col.Header, edit_id, col.Data, i, col.Header, i-1, table_columnCs[i].SelectableSrc, edit_id, edit_id, edit_id))

				table_columnCs[i].Filter = template.HTML(fmt.Sprintf(`
				<label class="form-label">%s:</label>
				<input
					id="%s"
					type="text"
					class="form-control dt-input dt-full-name typeahead-input"
					data-columnC="%d"
					placeholder="%s Text"
					data-columnC-index="%d" />

				<script>
				fetch('%s')
					.then(response => response.json())
					.then(data => {
						var prefetchExample = new Bloodhound({
							datumTokenizer: Bloodhound.tokenizers.whitespace,
							queryTokenizer: Bloodhound.tokenizers.whitespace,
							local: data // Use fetched data directly as the suggestion source
						});

						// Function to render default suggestions or search results
						function renderDefaults(q, sync) {
							if (q === '') {
								sync(prefetchExample.all()); // Show all suggestions when the query is empty
							} else {
								prefetchExample.search(q, sync); // Search based on the query
							}
						}

						// Initialize Typeahead on the input field
						$('#%s').typeahead(
							{
								hint: true,
								highlight: true,
								minLength: 0
							},
							{
								name: 'options',
								source: renderDefaults
							}
						);

						// Show all options when the input is focused and empty
						$('#%s').on('focus', function() {
							if (this.value === '') {
								$(this).typeahead('val', ''); // Clear the input to trigger default suggestions
								$(this).typeahead('open'); // Open the dropdown with all suggestions
							}
						});
						// Trigger a function when an option is selected from the dropdown
						$('#%s').on('typeahead:select', function(ev, suggestion) {
							// Perform an action here, e.g., trigger a keyup event, call a function, etc.
							$(this).trigger('keyup'); // Example: Trigger the keyup event
							filterColumnC($(this).attr('data-columnC'), $(this).val()); // Example: Trigger your filtering function
						});
					})
					.catch(error => console.error('Error fetching options data:', error));
				</script>
				  `, col.Header, filter_id, i, col.Header, i-1, table_columnCs[i].SelectableSrc, filter_id, filter_id, filter_id))
				table_columnCs[i].InsertField = template.HTML(fmt.Sprintf(`
				<label class="form-label">%s:</label>
				<input
					id="%s"
					type="text"
					name="%s"
					class="form-control"
					data-columnC="%d"
					placeholder="%s Text"
					data-columnC-index="%d" />

				<script>
				fetch('%s')
					.then(response => response.json())
					.then(data => {
						var prefetchExample = new Bloodhound({
							datumTokenizer: Bloodhound.tokenizers.whitespace,
							queryTokenizer: Bloodhound.tokenizers.whitespace,
							local: data
						});
						function renderDefaults(q, sync) {
							if (q === '') {
								sync(prefetchExample.all());
							} else {
								prefetchExample.search(q, sync);
							}
						}

						// Initialize Typeahead on the input field
						$('#%s').typeahead(
							{
								hint: true,
								highlight: true,
								minLength: 0
							},
							{
								name: 'options',
								source: renderDefaults
							}
						);

						$('#%s').on('focus', function() {
							if (this.value === '') {
								$(this).typeahead('val', '');
								$(this).typeahead('open');
							}
						});
						$('#%s').on('typeahead:select', function(ev, suggestion) {
							$(this).trigger('keyup');
							filterColumnC($(this).attr('data-columnC'), $(this).val());
						});
					})
					.catch(error => console.error('Error fetching options data:', error));
				</script>
				  `, col.Header, insert_id, col.Data, i, col.Header, i-1, table_columnCs[i].SelectableSrc, insert_id, insert_id, insert_id))

			} else {
				table_columnCs[i].InsertField = template.HTML(fmt.Sprintf(`
				<label class="form-label">%s:</label>
				<input
				  id="%s"
				  name="%s"
				  type="text"
				  class="form-control"
				  data-columnC="%d"
				  placeholder="%s Text"
				  data-columnC-index="%d" />`, col.Header, insert_id, col.Data, i, col.Header, i-1))

				table_columnCs[i].Filter = template.HTML(fmt.Sprintf(`<label class="form-label">%s:</label>
				<input
				  id="%s"
				  type="text"
				  class="form-control dt-input dt-full-name"
				  data-columnC="%d"
				  placeholder="%s Text"
				  data-columnC-index="%d" />`, col.Header, filter_id, i, col.Header, i-1))

				table_columnCs[i].EditForm = template.HTML(fmt.Sprintf(`<label class="form-label">%s</label>
				<input
				  id="%s"
				  type="text"
				  class="form-control"
				  name="%s"
				  data-columnC="%d"
				  placeholder="%s Text"
				  data-columnC-index="%d" />`, col.Header, edit_id, col.Data, i, col.Header, i-1))

			}
			className := "control"
			returnValue := ""
			if i > 0 {
				className = ""
				if editable {
					if table_columnCs[i].Editable {
						pass := ""
						if table_columnCs[i].Passwordable {
							pass = `pass="true"`
						}
						if table_columnCs[i].SelectableSrc != "" {
							returnValue = `<p class="selectable-suggestion" data-origin="'+extract_data+'" patch="` + endpoint + `" field="` + col.Data + `" select-option="` + string(table_columnCs[i].SelectableSrc) + `" point="'+full['id']+'" ` + pass + `>'+data+'</p>`
						} else {
							returnValue = `<p class="editable" data-origin="'+extract_data+'" patch="` + endpoint + `" field="` + col.Data + `" point="'+full['id']+'" ` + pass + `>'+data+'</p>`
						}
					} else {
						returnValue = `<p>'+data+'</p>`

					}
				} else {
					returnValue = `<p>'+data+'</p>`
				}
			}

			// table_columnCs[i].ColumnCConfig = template.JS(fmt.Sprintf(
			// 	`{
			// 		className: '%s',
			// 		targets: %d,
			// 		visible: %t,
			// 		orderable: %t,
			// 		render: function (data, type, full, meta) {
			// 		var extract_data = extractTxt_HTML(data);
			// 		return '%s';
			// 		}
			// 	},`, className, i, table_columnCs[i].Visible, table_columnCs[i].Orderable, returnValue))
			table_columnCs[i].ColumnCConfig.ClassName = className
			table_columnCs[i].ColumnCConfig.Targets = i
			table_columnCs[i].ColumnCConfig.Visible = table_columnCs[i].Visible
			table_columnCs[i].ColumnCConfig.Orderable = table_columnCs[i].Orderable
			table_columnCs[i].ColumnCConfig.Render = template.HTML(returnValue)

		case "image":
			// filter_id := "ft_" + table_name + "_" + col.Data
			edit_id := "ed_" + table_name + "_" + col.Data
			insert_id := "in_" + table_name + "_" + col.Data
			table_columnCs[i].EditId = edit_id
			// fmt.Println(filter_id)
			table_columnCs[i].InsertField = template.HTML(fmt.Sprintf(`
			<label class="form-label">%s:</label>
			<input
			  id="%s"
			  name="%s"
			  type="file"
			  class="form-control"
			  data-columnC="%d"
			  placeholder="Upload %s Image"
			  accept=".jpg, .jpeg, .png"
			  data-columnC-index="%d" />`, col.Header, insert_id, col.Data, i, col.Header, i-1))

			table_columnCs[i].Orderable = false
			table_columnCs[i].Filterable = false
			table_columnCs[i].Filter = ""

			table_columnCs[i].EditForm = template.HTML(fmt.Sprintf(`<label class="form-label">%s</label>
			<input
			  id="%s"
			  type="file"
			  class="form-control"
			  name="%s"
			  data-columnC="%d"
			  placeholder="Upload %s Image"
			  accept=".jpg, .jpeg, .png"
			  data-columnC-index="%d" />`, col.Header, edit_id, col.Data, i, col.Header, i-1))
			className := "control"
			returnValue := ""
			if i > 0 {
				className = ""
				if editable {
					if table_columnCs[i].Editable {
						returnValue = `<img src="'+data+'" alt="Image" style="width: 100%%;height:auto;" class="editable-image" data-origin="'+data+'" patch="` + endpoint + `" field="` + col.Data + `" point="'+full['id']+'" /> `
					} else {
						returnValue = `<img src="'+data+'" alt="Image" style="width: 100%% ; height: auto;"/>`

					}
				} else {
					returnValue = `<img src="'+data+'" alt="Image" style="width: 100%% ; height: auto;"/>`
				}
			}

			table_columnCs[i].ColumnCConfig.ClassName = className
			table_columnCs[i].ColumnCConfig.Targets = i
			table_columnCs[i].ColumnCConfig.Visible = table_columnCs[i].Visible
			table_columnCs[i].ColumnCConfig.Orderable = table_columnCs[i].Orderable
			table_columnCs[i].ColumnCConfig.Render = template.HTML(fmt.Sprintf(`<div style="width: 50px;height: 50px;overflow: hidden;">%s</div>`, returnValue))

		case "time.Time":
			filter_id := "ft_" + table_name + "_" + col.Data
			edit_id := "ed_" + table_name + "_" + col.Data
			insert_id := "in_" + table_name + "_" + col.Data
			table_columnCs[i].EditId = edit_id
			table_columnCs[i].InsertField = template.HTML(fmt.Sprintf(`
			<label class="form-label">%s:</label>
			<input
			  id="%s"
			  name="%s"
			  type="text"
			  class="form-control flatpickr-datetime"
			  data-columnC="%d"
			  placeholder="%s YYYY-MM-DD HH:MM"
			  data-columnC-index="%d" />`, col.Header, insert_id, col.Data, i, col.Header, i-1))

			table_columnCs[i].EditForm = template.HTML(fmt.Sprintf(`<label class="form-label">%s</label>
			<input
			  id="%s"
			  type="number"
			  class="form-control flatpickr-datetime"
			  name="%s"
			  data-columnC="%d"
			  placeholder="%s YYYY-MM-DD HH:MM"
			  data-columnC-index="%d" />`, col.Header, edit_id, col.Data, i, col.Header, i-1))

			table_columnCs[i].Filter = template.HTML(fmt.Sprintf(`<label class="form-label">%s:</label>
			<div class="mb-0">
			  <input
			  	id="%s"
				type="text"
				class="form-control dt-date flatpickr-range dt-input"
				data-columnC="%d"
				placeholder="StartDate to EndDate"
				data-columnC-index="%d"
				name="dt_date" />
			  <input
				type="hidden"
				class="form-control dt-date start_date_%s dt-input"
				data-columnC="%d"
				data-columnC-index="%d"
				name="value_from_start_date" />
			  <input
				type="hidden"
				class="form-control dt-date end_date_%s dt-input"
				name="value_from_end_date"
				data-columnC="%d"
				data-columnC-index="%d" />
			</div>`, col.Header, filter_id, i, i-1, table_name, i, i-1, table_name, i, i-1))
		case "int", "int8", "int16", "int32", "uint", "int64":
			filter_id := "ft_" + table_name + "_" + col.Data
			edit_id := "ed_" + table_name + "_" + col.Data
			insert_id := "in_" + table_name + "_" + col.Data
			table_columnCs[i].EditId = edit_id
			if table_columnCs[i].SelectableSrc != "" {
				table_columnCs[i].EditForm = template.HTML(fmt.Sprintf(`
				<label class="form-label">%s</label>
				<input
					id="%s"
					type="number"
					class="form-control"
					name="%s"
					data-columnC="%d"
					placeholder="%s Number"
					data-columnC-index="%d" />

				<script>
				fetch('%s')
					.then(response => response.json())
					.then(data => {
						var prefetchExample = new Bloodhound({
							datumTokenizer: Bloodhound.tokenizers.whitespace,
							queryTokenizer: Bloodhound.tokenizers.whitespace,
							local: data // Use fetched data directly as the suggestion source
						});

						// Function to render default suggestions or search results
						function renderDefaults(q, sync) {
							if (q === '') {
								sync(prefetchExample.all()); // Show all suggestions when the query is empty
							} else {
								prefetchExample.search(q, sync); // Search based on the query
							}
						}

						// Initialize Typeahead on the input field
						$('#%s').typeahead(
							{
								hint: true,
								highlight: true,
								minLength: 0
							},
							{
								name: 'options',
								source: renderDefaults
							}
						);

						// Show all options when the input is focused and empty
						$('#%s').on('focus', function() {
							if (this.value === '') {
								$(this).typeahead('val', ''); // Clear the input to trigger default suggestions
								$(this).typeahead('open'); // Open the dropdown with all suggestions
							}
						});
						// Trigger a function when an option is selected from the dropdown
						$('#%s').on('typeahead:select', function(ev, suggestion) {
							// Perform an action here, e.g., trigger a keyup event, call a function, etc.
							$(this).trigger('keyup'); // Example: Trigger the keyup event
							filterColumnC($(this).attr('data-columnC'), $(this).val()); // Example: Trigger your filtering function
						});
					})
					.catch(error => console.error('Error fetching options data:', error));
				</script>
				  `, col.Header, edit_id, col.Data, i, col.Header, i-1, table_columnCs[i].SelectableSrc, edit_id, edit_id, edit_id))

				table_columnCs[i].Filter = template.HTML(fmt.Sprintf(`
				<label class="form-label">%s:</label>
				<input
					id="%s"
					type="number"
					class="form-control dt-input dt-full-name typeahead-input"
					data-columnC="%d"
					placeholder="%s Number"
					data-columnC-index="%d" />

				<script>
				fetch('%s')
					.then(response => response.json())
					.then(data => {
						var prefetchExample = new Bloodhound({
							datumTokenizer: Bloodhound.tokenizers.whitespace,
							queryTokenizer: Bloodhound.tokenizers.whitespace,
							local: data // Use fetched data directly as the suggestion source
						});

						// Function to render default suggestions or search results
						function renderDefaults(q, sync) {
							if (q === '') {
								sync(prefetchExample.all()); // Show all suggestions when the query is empty
							} else {
								prefetchExample.search(q, sync); // Search based on the query
							}
						}

						// Initialize Typeahead on the input field
						$('#%s').typeahead(
							{
								hint: true,
								highlight: true,
								minLength: 0
							},
							{
								name: 'options',
								source: renderDefaults
							}
						);

						// Show all options when the input is focused and empty
						$('#%s').on('focus', function() {
							if (this.value === '') {
								$(this).typeahead('val', ''); // Clear the input to trigger default suggestions
								$(this).typeahead('open'); // Open the dropdown with all suggestions
							}
						});
						// Trigger a function when an option is selected from the dropdown
						$('#%s').on('typeahead:select', function(ev, suggestion) {
							// Perform an action here, e.g., trigger a keyup event, call a function, etc.
							$(this).trigger('keyup'); // Example: Trigger the keyup event
							filterColumnC($(this).attr('data-columnC'), $(this).val()); // Example: Trigger your filtering function
						});
					})
					.catch(error => console.error('Error fetching options data:', error));
				</script>
				  `, col.Header, filter_id, i, col.Header, i-1, table_columnCs[i].SelectableSrc, filter_id, filter_id, filter_id))
				table_columnCs[i].InsertField = template.HTML(fmt.Sprintf(`
				<label class="form-label">%s:</label>
				<input
					id="%s"
					type="number"
					name="%s"
					class="form-control"
					data-columnC="%d"
					placeholder="%s Number"
					data-columnC-index="%d" />

				<script>
				fetch('%s')
					.then(response => response.json())
					.then(data => {
						var prefetchExample = new Bloodhound({
							datumTokenizer: Bloodhound.tokenizers.whitespace,
							queryTokenizer: Bloodhound.tokenizers.whitespace,
							local: data
						});
						function renderDefaults(q, sync) {
							if (q === '') {
								sync(prefetchExample.all());
							} else {
								prefetchExample.search(q, sync);
							}
						}

						// Initialize Typeahead on the input field
						$('#%s').typeahead(
							{
								hint: true,
								highlight: true,
								minLength: 0
							},
							{
								name: 'options',
								source: renderDefaults
							}
						);

						$('#%s').on('focus', function() {
							if (this.value === '') {
								$(this).typeahead('val', '');
								$(this).typeahead('open');
							}
						});
						$('#%s').on('typeahead:select', function(ev, suggestion) {
							$(this).trigger('keyup');
							filterColumnC($(this).attr('data-columnC'), $(this).val());
						});
					})
					.catch(error => console.error('Error fetching options data:', error));
				</script>
				  `, col.Header, insert_id, col.Data, i, col.Header, i-1, table_columnCs[i].SelectableSrc, insert_id, insert_id, insert_id))

			} else {
				table_columnCs[i].InsertField = template.HTML(fmt.Sprintf(`
				<label class="form-label">%s:</label>
				<input
				  id="%s"
				  name="%s"
				  type="number"
				  class="form-control"
				  data-columnC="%d"
				  placeholder="%s number"
				  data-columnC-index="%d" />`, col.Header, insert_id, col.Data, i, col.Header, i-1))

				table_columnCs[i].Filter = template.HTML(fmt.Sprintf(`<label class="form-label">%s:</label>
				  <input
					id="%s"
					type="number"
					class="form-control dt-input dt-full-name"
					data-columnC="%d"
					placeholder="%s Text"
					data-columnC-index="%d" />`, col.Header, filter_id, i, col.Header, i-1))

				table_columnCs[i].EditForm = template.HTML(fmt.Sprintf(`<label class="form-label">%s</label>
				<input
				  id="%s"
				  type="number"
				  class="form-control"
				  name="%s"
				  data-columnC="%d"
				  placeholder="%s Number"
				  data-columnC-index="%d" />`, col.Header, edit_id, col.Data, i, col.Header, i-1))

			}

			className := "control"
			returnValue := ""
			if i > 0 {
				className = ""
				if editable {
					if table_columnCs[i].Editable {
						if table_columnCs[i].SelectableSrc != "" {
							returnValue = `<p class="selectable-suggestion" data-origin="'+data+'" patch="` + endpoint + `" field="` + col.Data + `" select-option="` + string(table_columnCs[i].SelectableSrc) + `" point="'+full['id']+'" >'+data+'</p>`
						} else {
							returnValue = `<p class="editable" data-origin="'+data+'" patch="` + endpoint + `" field="` + col.Data + `" point="'+full['id']+'" >'+data+'</p>`
						}
					} else {
						returnValue = `<p>'+data+'</p>`

					}
				} else {
					returnValue = `<p>'+data+'</p>`
				}
			}

			table_columnCs[i].ColumnCConfig.ClassName = className
			table_columnCs[i].ColumnCConfig.Targets = i
			table_columnCs[i].ColumnCConfig.Visible = table_columnCs[i].Visible
			table_columnCs[i].ColumnCConfig.Orderable = table_columnCs[i].Orderable
			table_columnCs[i].ColumnCConfig.Render = template.HTML(returnValue)

		default:
			table_columnCs[i].Filterable = false
		}

	}
	actionable := ""
	if editable || deletable {
		table_columnCs = append(table_columnCs, ColumnC{Data: "", Header: template.HTML("<i class='bx bx-run'></i>"), Type: "", Editable: false})
		actionable = "orderable"
	}

	// fmt.Println("show_header")
	// fmt.Println(!hideHeader)
	export_copy := false
	export_print := false
	export_pdf := false
	export_csv := false
	export_all_csv := false
	for _, export_type := range exportType {
		switch export_type {
		case EXPORT_COPY:
			export_copy = true
		case EXPORT_PRINT:
			export_print = true
		case EXPORT_CSV:
			export_csv = true
		case EXPORT_PDF:
			export_csv = true
		case EXPORT_ALL:
			export_all_csv = true
		}
	}
	passtrue := ""
	if passwordable {
		passtrue = `pass="true"`

	}
	renderedHTML, err := RenderTemplateToString("table.html", map[string]any{
		"title":           template.HTML(title),
		"table_name":      table_name,
		"endpoint":        template.URL(endpoint),
		"table_columnCs":  table_columnCs,
		"actionable":      actionable,
		"insertable":      insertable,
		"page_length":     page_length,
		"length_menu":     length_menu,
		"order":           order,
		"hide_header":     hideHeader,
		"passwordable":    passwordable,
		"passtrue":        passtrue,
		"export_copy":     export_copy,
		"export_print":    export_print,
		"export_pdf":      export_pdf,
		"export_csv":      export_csv,
		"export_all_csv":  export_all_csv,
		"scrollUpDown":    scrollUpDown,
		"scrollLeftRight": scrollLeftRight,
		"columnC_array":   columnC_array,
	})
	if err != nil {
		logrus.Error(err)
		fmt.Println("Error rendering template:", err)
		return template.HTML("Error rendering template")
	}

	return template.HTML(renderedHTML)
}
