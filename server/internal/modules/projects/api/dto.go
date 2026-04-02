package api

import "camopanel/server/internal/modules/projects/usecase"

func toProjectResponse(view usecase.ProjectView) usecase.ProjectView {
	return view
}
