package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ModelHasRole struct{ ent.Schema }

func (ModelHasRole) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "model_has_roles"},
	}
}

func (ModelHasRole) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id").Unique().Immutable(),
		field.Uint("role_id"),
		field.String("model_type").MaxLen(255).NotEmpty(),
		field.Uint("model_id"),
		field.String("tenant_id").MaxLen(100).Optional().Nillable(),
	}
}

func (ModelHasRole) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("role_id", "model_type", "model_id").Unique().
			StorageKey("idx_model_roles_unique"),
		index.Fields("tenant_id").StorageKey("idx_model_roles_tenant"),
	}
}
