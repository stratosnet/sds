$.fn.addRow = function(data) {
    tr = "<tr>"
    $.each(data, function(index, item) {
        tr += "<td>"+item+"</td>"
    })
    tr += "</tr>"
    $(this).append(tr)
}