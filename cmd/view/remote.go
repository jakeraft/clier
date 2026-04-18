package view

import remoteapi "github.com/jakeraft/clier/internal/adapter/api"

func ResourceOf(resp *remoteapi.ResourceResponse) remoteapi.ResourceResponse {
	out := *resp
	if out.Refs == nil {
		out.Refs = []remoteapi.ResolvedRef{}
	}
	return out
}

func ResourceListOf(resp *remoteapi.ListResponse) remoteapi.ListResponse {
	out := remoteapi.ListResponse{
		Items: make([]remoteapi.ResourceResponse, 0, len(resp.Items)),
		Total: resp.Total,
	}
	for i := range resp.Items {
		item := resp.Items[i]
		if item.Refs == nil {
			item.Refs = []remoteapi.ResolvedRef{}
		}
		out.Items = append(out.Items, item)
	}
	return out
}

func OrgOf(resp *remoteapi.OrgResponse) remoteapi.OrgResponse {
	return *resp
}
