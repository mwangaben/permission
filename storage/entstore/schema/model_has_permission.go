package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ModelHasPermission struct{ ent.Schema }

func (ModelHasPermission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "model_has_permissions"},
	}
}

func (ModelHasPermission) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id").Unique().Immutable(),
		field.Uint("permission_id"),
		field.String("model_type").MaxLen(255).NotEmpty(),
		field.Uint("model_id"),
		field.String("tenant_id").MaxLen(100).Optional().Nillable(),
	}
}

func (ModelHasPermission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("permission_id", "model_type", "model_id").Unique().
			StorageKey("idx_model_permissions_unique"),
		index.Fields("tenant_id").StorageKey("idx_model_permissions_tenant"),
	}
}
